package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net"
	"os"
	"strconv"
	"strings"

	"github.com/tidwall/gjson"
)

var (
	serverAddr               = "127.0.0.1:9988"
	clientIndex        int64 = 0
	clientMaps               = map[int64]net.Conn{}
	clientGroups             = map[string]map[int64]net.Conn{}
	clientGroupsMaster       = map[string]map[int64]net.Conn{}
)

type message struct {
	From   string //`slave/master`
	To     string //`slave/master/allslave`
	Group  string //`group_name`
	Action string //`join,chat`
	Msg    string //`json`
	Role   string //`slave/master/system`
}

func main() {
	fmt.Println(strconv.ParseInt("11123", 10, 64))
	initServer()
}

func initServer() {
	addr, err := net.ResolveTCPAddr("tcp", serverAddr)
	checkError(err)
	listerner, err := net.ListenTCP("tcp", addr)
	checkError(err)
	defer listerner.Close()

	for {
		client, err := listerner.Accept()
		if err != nil {
			continue
		}
		clientIndex++
		Log(client.RemoteAddr().String(), " tcp connect success", clientIndex)
		go handleConn(client, clientIndex)
	}
}

func handleConn(conn net.Conn, index int64) {
	clientMaps[index] = conn
	defer func() {
		conn.Close()
		delete(clientMaps, index)
	}()

	buffer := make([]byte, 10240)
	group := ""
	role := ""
	for {
		n, err := conn.Read(buffer)
		if err != nil {
			if err == io.EOF {
				if group != "" {
					delete(clientGroups[group], index)
				}
				if role == "master" {
					delete(clientGroupsMaster[group], index)
				}
				Log(conn.RemoteAddr().String(), " disconnection ")
				return
			}
			Log(conn.RemoteAddr().String(), " connection error: ", err)
			return
		}

		// Log(conn.RemoteAddr().String(), "receive data length:", n)
		// Log(conn.RemoteAddr().String(), "receive data string:", string(buffer[:n]))
		receive := string(buffer[:n])
		//使用换行解决消息粘包
		msgsArr := strings.Split(receive, "\r\n")
		fmt.Println(receive)
		fmt.Println(msgsArr)
		for _, msg := range msgsArr {
			if msg == "" {
				continue
			}
			if !gjson.Valid(msg) {
				Send(conn, message{"system", "", "", "error", "use json format", "system"})
				continue
			}
			switch gjson.Get(msg, "Action").String() {
			case "join":
				group = gjson.Get(msg, "Group").String()
				role = gjson.Get(msg, "Role").String()

				temp_items := clientGroups[group]
				if len(temp_items) == 0 {
					temp_items = make(map[int64]net.Conn)
				}
				temp_items[index] = conn
				clientGroups[group] = temp_items

				if role == "master" {
					temp_items = clientGroupsMaster[group]
					if len(temp_items) == 0 {
						temp_items = make(map[int64]net.Conn)
					}
					temp_items[index] = conn
					clientGroupsMaster[group] = temp_items
				}
				Send(conn, message{"system", "", "", "join", fmt.Sprintf("{\"ClientId\":%v}", index), "system"})
			case "chat":
				Chat(msg)
			}
		}
	}
}

func Chat(data string) {
	from := gjson.Get(data, "From").String()
	to := gjson.Get(data, "To").String()
	group := gjson.Get(data, "Group").String()
	msg := gjson.Get(data, "Msg").String()

	//主 -> 从群
	if from == "master" && to == "allslave" {
		for _, c := range clientGroups[group] {
			Send(c, message{from, to, group, "chat", msg, "master"})
		}
	}

	// 从 -> 主群
	if from == "slave" && to == "master" {
		for _, c := range clientGroupsMaster[group] {
			Send(c, message{from, to, group, "chat", msg, "slave"})
		}
	}

	// 点对点
	clientId, err := strconv.ParseInt(to, 10, 64)
	if err != nil {
		c := clientMaps[clientId]
		Send(c, message{from, to, group, "chat", msg, "slave"})
	}

}

func Send(c net.Conn, msg message) {
	json_str, err := json.Marshal(msg)
	if err != nil {
		fmt.Println(err.Error())
		return
	}
	c.Write(bytes.Join([][]byte{json_str, []byte("\r\n")}, []byte{}))
}

func checkError(err error) {
	if err != nil {
		fmt.Println(err.Error())
		os.Exit(1)
	}
}

func Log(v ...interface{}) {
	fmt.Println(v...)
}
