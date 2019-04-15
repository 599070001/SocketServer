# SocketServer
golang socket server

# 消息结构体
```
type message struct {
	From   string //`slave/master`
	To     string //`slave/master/allslave`
	Group  string //`group_name`
	Action string //`join,chat`
	Msg    string //`json`
	Role   string //`slave/master/system`
}
```
#  step1
join

{"From":"slave","To":"","Group":"房间1","Action":"join","Msg":"","Role":"slave"}

# step2
chat

{"From":"slave","To":"master","Group":"房间1","Action":"chat","Msg":"{}","Role":"slave"}


