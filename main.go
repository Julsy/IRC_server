package main

import (
	"container/list"
	"flag"
	"fmt"
	"net"
	"strings"
	// "errors"
)

const server_pass string = "supersecret"

type ClientChat struct {
	Name        *string
	In          chan string
	Out         chan string
	Conn        net.Conn
	Quit        chan bool
	ListChain   *list.List
	ListChannel *list.List
	Auth        bool
	LoggedIn    bool
}

type ChannelChat struct {
	Name      string
	Topic     string
	UsersList *list.List
}

var passwords = map[string]string{}

var command_list = map[string]func(*ClientChat, []string){
	"NICK":     cmd_NICK,
	"USER":     cmd_USER,
	"JOIN":     cmd_JOIN,
	"REGISTER": cmd_REGISTER,
	"LOGIN":    cmd_LOGIN,
	// "PRIVMSG":  cmd_PRIVMSG,
	// "PING":	cmd_PING,
	"QUIT": cmd_QUIT,
	"PASS": cmd_PASS,
	// "KICK": cmd_KICK, // Parameters: <channel> <user> [<comment>]
	// "PART": cmd_PART,
	// "LIST": cmd_LIST,
	// "TOPIC": cmd_TOPIC,
}

/*
** Connection for some reason fails when unexpected quit of a client
** Due to removing a user who already is gone.
**
		panic: interface conversion: interface {} is main.ClientChat, not main.ChannelChat

		goroutine 10 [running]:
		main.(*ClientChat).delete(0xc420080500)
			/Users/julsy/Work/Study/GO/Frozen/client.go:33 +0x3e9
		main.(*ClientChat).Close(0xc420080500)
			/Users/julsy/Work/Study/GO/Frozen/client.go:20 +0x70
		main.cmd_QUIT(0xc420080500, 0xc42005a080, 0x6, 0x6)
			/Users/julsy/Work/Study/GO/Frozen/main.go:95 +0x2b
		main.client_recv(0xc420080500)
			/Users/julsy/Work/Study/GO/Frozen/main.go:191 +0x39d
		created by main.clientHandle
			/Users/julsy/Work/Study/GO/Frozen/main.go:247 +0x285
		exit status 2
**
*/

func cmd_LOGIN(client *ClientChat, params []string) {
	if len(params) != 2 {
		client.sendmsg("", "461", "LOGIN", ":Not enough parameters")
		return
	}
	if val, ok := passwords[params[0]]; ok {
		if params[1] == val {
			fmt.Printf("Logged in as %s (%s)\n", params[0], val)
		} else {
			client.sendmsg("", "464", ":Password incorrect")
		}
	} else {
		client.sendmsg("", "451", ":You have not registered")
	}
}

func cmd_REGISTER(client *ClientChat, params []string) {
	if len(params) != 2 {
		client.sendmsg("", "461", "REGISTER", ":Not enough parameters")
		return
	}
	if _, ok := passwords[params[0]]; ok {
		client.sendmsg("", "462", "You may not reregister")
		return
	}
	passwords[params[0]] = params[1]
	fmt.Printf("User %s set password to %s\n", params[0], passwords[params[0]])
}

func cmd_PASS(client *ClientChat, params []string) {
	if len(params) == 0 {
		client.sendmsg("", "461", "PASS", ":Not enough parameters")
		return
	}
	if params[0] == server_pass {
		client.Auth = true
		return
	}
	client.sendmsg("", "464", "ERR_PASSWDMISMATCH", ":Password incorrect")
}

func cmd_QUIT(client *ClientChat, params []string) {
	client.Close()
	fmt.Printf("Bye felicia\n")
}

func cmd_NICK(client *ClientChat, params []string) {
	if len(params) == 0 {
		client.sendmsg("", "431", "No nickname given")
		return
	}
	for i := client.ListChain.Front(); i != nil; i = i.Next() {
		c := i.Value.(ClientChat)
		if *c.Name == params[0] {
			client.sendmsg("", "433", *client.Name, ":Nickname is already in use")
			fmt.Printf("Nick is already taken: %s \n", params[0])
			return
		}
	}
	*client.Name = params[0]
	fmt.Printf("Nick to set %s (%s)\n", *client.Name, params[0])
}

func cmd_USER(client *ClientChat, params []string) {
	client.sendmsg("", "375", *client.Name, "-:- Message of the day - ")
	client.sendmsg("", "372", *client.Name, ":- We da cooliest")
	client.sendmsg("", "376", *client.Name, ":End of /MOTD command")
}

func list_channels(client *ClientChat) {
	fmt.Println("******************* Listing all Users ***************")
	for i := client.ListChannel.Front(); i != nil; i = i.Next() {
		ch := i.Value.(ChannelChat)
		fmt.Printf("Channel = %s\n", ch.Name)
	}
	fmt.Println("*****************************************************")
}

func cmd_JOIN(client *ClientChat, params []string) {
	channel := client.channel_add(params[0])
	channel.adduser(client)
	client.sendmsg("", "332", *client.Name, ":", channel.Topic)
	send_list(client, channel)
	for i := channel.UsersList.Front(); i != nil; i = i.Next() {
		c := i.Value.(ClientChat)
		if *c.Name == *client.Name {
			continue
		}
		send_list(&c, channel)
	}
}

func (c *ClientChat) nick_open(nickname string) bool {
	for i := c.ListChain.Front(); i != nil; i = i.Next() {
		if *i.Value.(ClientChat).Name == nickname {
			return true
		}
	}
	return false
}

func list_all(client *ClientChat) {
	fmt.Println("=================== Listing all Users ===============")
	for i := client.ListChain.Front(); i != nil; i = i.Next() {
		c := i.Value.(ClientChat)
		fmt.Printf("(%p) %s - %s\n", &c, c.Conn.LocalAddr().String(), *c.Name)
	}
	fmt.Println("=====================================================")
}

func client_recv(client *ClientChat) {
	buf := make([]byte, 0xFFFF)

	fmt.Println("client_recv(): start for: ", *client.Name)
	for {
		_, err := client.Conn.Read(buf)
		if err != nil {
			fmt.Println("Err", err)
			break
		}
		for _, msg := range strings.Split(string(buf), "\r\n") {
			fmt.Println("GET: ", msg)
			command := strings.ToUpper(strings.Split(msg, " ")[0])
			if client.Auth != true && command != "PASS" {
				fmt.Println("Authenticate dumbass")
				continue
			}
			params := strings.Split(msg, " ")[1:]
			for i := client.ListChain.Front(); i != nil; i = i.Next() {
				c := i.Value.(ClientChat)
				if strings.ToUpper(*c.Name) == command[1:] {
					cmd_NICK(&c, params[1:])
					break
				}
				list_all(client)
			}
			cmd, found := command_list[command]
			if found {
				cmd(client, params)
				list_all(client)
				list_channels(client)
			} else {
				fmt.Println("MSG: Unknown Message >>>", msg)
			}
		}
		// Clear it
		for i := 0; i < 0xFFFF; i++ {
			buf[i] = 0x0
		}
	}
	fmt.Println("clientreceiver(): stop for: ", client.Name)
}

func send_list(client *ClientChat, ch *ChannelChat) {
	namereply := ""
	for i := ch.UsersList.Front(); i != nil; i = i.Next() {
		c := i.Value.(ClientChat)
		namereply += *c.Name + " "
	}
	client.sendmsg("", "353", ch.Name, fmt.Sprintf(":%s", namereply))
	// client.sendmsg("", "353", *client.Name, "=", *client.Name, fmt.Sprintf(":%s", namereply))
	client.sendmsg("", "366", ch.Name, ":End of /NAMES list")
}

func client_send(client *ClientChat) {
	for {
		select {
		case buf := <-client.In:
			client.Conn.Write([]byte(buf + "\r\n"))
		case <-client.Quit:
			break
		}
	}
	fmt.Printf("client_send() exited.\n")
}

func clientHandle(conn net.Conn, userList *list.List, channellist *list.List) {
	var name string

	newclient := &ClientChat{
		&name,
		make(chan string),
		make(chan string),
		conn,
		make(chan bool),
		userList,
		channellist,
		false,
		false,
	}
	userList.PushBack(*newclient)
	c := userList.Back().Value.(ClientChat)

	go client_send(&c)
	go client_recv(&c)
	list_all(newclient)
}

func main() {
	flag.Parse()
	fmt.Println("Starting IRC Server!")
	clientlist := list.New()
	channellist := list.New()
	listener, err := net.Listen("tcp", "0.0.0.0:1337")
	if err != nil {
		fmt.Println("Failed to bind: ", err)
		return
	}
	defer listener.Close()
	for {
		conn, err := listener.Accept()
		if err != nil {
			fmt.Println(err)
			return
		}
		go clientHandle(conn, clientlist, channellist)
	}
}
