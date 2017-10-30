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
	Name       string
	Topic      string
	Visible    *int
	UsersList  *list.List
	Moderators []string
}

var passwords = map[string]string{}

var command_list = map[string]func(*ClientChat, []string){
	"NICK":    cmd_NICK,
	"USER":    cmd_USER,
	"JOIN":    cmd_JOIN,
	"LOGIN":   cmd_LOGIN,
	"PRIVMSG": cmd_PRIVMSG,
	"QUIT":    cmd_QUIT,
	"PASS":    cmd_PASS,
	"KICK":    cmd_KICK, // Parameters: <channel> <user>
	"LIST":    cmd_LIST,
	"UPASS":	cmd_UPASS,
	"NAMES":	cmd_NAMES,
	// "TOPIC": cmd_TOPIC,
	"PING":	cmd_PING,
	"PART": cmd_PART,
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

** #2nd bug
** Empty message panics, sanitize inputs.

** 3rd Set a Topic

!!!
!!!	4th Fix formatting messages to client
!!! You have to be able to use any IRC client to connect and test the
!!! functionality of your server.

PREFIX - SERVER NAME???
**How to connect normally **
telnet irc.freenode.net 6667
NICK randomuser29
USER randomuser29 * * :TestMe
** end how to connect **
**
*/

func cmd_NAMES(client *ClientChat, params []string) {
	totalnames := ""
	for i := client.ListChain.Front(); i != nil; i = i.Next() {
		c := i.Value.(ClientChat)
		totalnames = totalnames + *c.Name + " "
	}
	client.sendmsg("", "353", totalnames);
	client.sendmsg("", "366", ":End of /NAMES list")
}

/*
** When a client quits immediately, we need to remove it from all the channels it was connnected to
** This can happen from someone Force Killing their IRC client!!!
*/

func cmd_LOGIN(client *ClientChat, params []string) {
	if len(params) != 2 {
		client.sendmsg("", "461", "LOGIN", ":Not enough parameters")
		return
	}
	if val, ok := passwords[params[0]]; ok {
		if params[1] == val {
			fmt.Printf("Logged in as %s (%s)\n", params[0], val)
			client.LoggedIn = true;
		} else {
			client.sendmsg("", "464", ":Password incorrect")
		}
	} else {
		client.sendmsg("", "451", ":You have not registered")
	}
}

/*
** UPASS <password> <newpassword>
*/

func cmd_UPASS(client *ClientChat, params []string) {
	if len(params) != 1 {
		client.sendmsg("", "461", "UPASS", ":Not enough parameters")
		return
	}
	if *client.Name != "" {
		if val, ok := passwords[*client.Name];ok {
			if params[0] == val && client.LoggedIn && len(params) >= 2 {
				passwords[*client.Name] = params[1];
				fmt.Printf("Set new pass as %s (%s)\n", params[1], passwords[*client.Name])
			} else {
				client.sendmsg("", "464", ":Password incorrect")
			}
		} else {
			passwords[*client.Name] = params[0];
		}
	} else {
		fmt.Printf("empty username? (\"%s\"", *client.Name);
		client.sendmsg("", "432", *client.Name, ":Erroneus nickname");
    // 431     ERR_NONICKNAMEGIVEN
    //                     ":No nickname given"
	}
}

func cmd_PING(client *ClientChat, params []string) {
	client.sendmsg("", "PONG")
}

func cmd_PART(client *ClientChat, params []string) {
	if len(params) == 0 {
		client.sendmsg("", "461", "PART", ":Not enough parameters")
		return
	}
	if params[0][0] == '#' {
		channel := client.channel_add(params[0]);
		channel.deluser(client);
	}
}

func cmd_USER(client *ClientChat, params []string) {
	if len(params) < 4 {
		client.sendmsg("", "461", "REGISTER", ":Not enough parameters")
		return
	}

	// 462     ERR_ALREADYREGISTRED  ":You may not reregister"

	if _, ok := passwords[params[0]]; ok {
		client.sendmsg("", "462", "You may not reregister")
		return
	}
	// passwords[params[0]] = params[1]
	// fmt.Printf("User %s set password to %s\n", params[0], passwords[params[0]])
	client.sendmsg("", "375", *client.Name, "-:- Message of the day - ")
	client.sendmsg("", "372", *client.Name, ":- We da cooliest")
	client.sendmsg("", "376", *client.Name, ":End of /MOTD command")
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
	client.sendmsg("", "QUIT", ":Quit");
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

func cmd_LIST(client *ClientChat, params []string) {
	// client.In <- "Channel :Users  Name"
	if client.LoggedIn == false {
		client.sendmsg("", "444", *client.Name, ":User not logged in")
		return ;
	}
	client.sendmsg("", "321", "*", "LIST", "Channel :Users  Name")
	for i := client.ListChannel.Front(); i != nil; i = i.Next() {
		ch := i.Value.(ChannelChat)
		client.sendmsg("", "322", "*", fmt.Sprintf("%s %d :%s", ch.Name, *ch.Visible, ch.Topic))
		fmt.Printf("Channel = %s\n", ch.Name)
	}
	client.sendmsg("", "323", "*", ":End of /LIST")
}

func list_channels(client *ClientChat) {
	fmt.Println("**************** Listing all Channels ***************")
	for i := client.ListChannel.Front(); i != nil; i = i.Next() {
		ch := i.Value.(ChannelChat)
		fmt.Printf("Channel = %s (%d)\n", ch.Name, *ch.Visible)
	}
	fmt.Println("*****************************************************")
}

// FIX ME
/*
**
:Julia!~Julia@192.168.0.103 JOIN :#channel
:192.168.0.103 353 Julia = #channel :Julia 
:192.168.0.103 366 Julia #channel :End of /NAMES list.
:192.168.0.103 332 Julia #channel :
:192.168.0.103 353 Julia = #channel :Julia 
:192.168.0.103 366 Julia #channel :End of /NAMES list.

vs

:Julia!~Julia@ JOIN :#channel
: 332 Julia #channel :
: 353 #channel :Julia 
: 366 #channel :End of /NAMES list

*/
func cmd_JOIN(client *ClientChat, params []string) {
	c := params[0];
	if c[0] != '#' {
		c = "#" + c;
	}
	if client.LoggedIn == false {
		client.sendmsg("", "444", *client.Name, ":User not logged in")
		return ;
	}
	channel := client.channel_add(c)
	if is_inchannel(channel, client) == false {
		channel.adduser(client)
		client.sendmsg(fmt.Sprintf("%s!~%s@", *client.Name, *client.Name), "JOIN", fmt.Sprintf(":%s", channel.Name))
		fmt.Printf("Channel has visible %d\n", *channel.Visible)
		client.sendmsg("", "332", *client.Name, channel.Name, fmt.Sprintf(":%s", channel.Topic))
		for i := channel.UsersList.Front(); i != nil; i = i.Next() {
			c := i.Value.(ClientChat)
			send_list(&c, channel)
		}
	}
}

func is_inchannel(channel *ChannelChat, client *ClientChat) bool {
	for i := channel.UsersList.Front(); i != nil; i = i.Next() {
		c := i.Value.(ClientChat)
		if *c.Name == *client.Name {
			c.sendmsg("", "443", *c.Name, channel.Name, ":is already on channel")
			return (true)
		}
	}
	return (false)
}

func is_moderator(channel *ChannelChat, client *ClientChat) bool {
	for _, err := range channel.Moderators {
		if err == *client.Name {
			return (true)
		}
	}
	return (false)
}

func cmd_KICK(client *ClientChat, params []string) {
	if len(params) == 0 {
		client.sendmsg("", "461", "PASS", ":Not enough parameters")
		return
	}
	if client.LoggedIn == false {
		client.sendmsg("", "444", *client.Name, ":User not logged in")
		return ;
	}
	for i := client.ListChannel.Front(); i != nil; i = i.Next() {
		c := i.Value.(ChannelChat)
		if c.Name == params[0] {
			if is_moderator(&c, client) {
				for j := c.UsersList.Front(); j != nil; j = j.Next() {
					u := j.Value.(ClientChat)
					if *u.Name == params[1] {
						fmt.Printf("User %s was kicked from %s by %s", *u.Name, c.Name, *client.Name)
						c.deluser(&u)
					}
				}
			}
		}
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


func (ch *ChannelChat) chan_msg(client *ClientChat, msg string) {
	for i := ch.UsersList.Front(); i != nil; i = i.Next() {
		c := i.Value.(ClientChat);
		//Can't send to himself
		if *c.Name != *client.Name {
			c.sendmsg(fmt.Sprintf("%s!%s@%s", *client.Name, *client.Name, client.Conn.RemoteAddr()), "PRIVMSG", ch.Name, msg);
		}
	}
}

/*
** So this apparently handles both PRIVATE messages and channel messages? tf?
** w0w
**
*/
func cmd_PRIVMSG(client *ClientChat, params []string) {
	if len(params) < 2 {
		client.sendmsg("", "461", "PRIVMSG", ":Not enough parameters")
		return
	}
	if client.LoggedIn == false {
		client.sendmsg("", "444", *client.Name, ":User not logged in")
		return ;
	}
	receiver := params[0];
	if receiver[0] == '#' {
		channel := client.channel_add(receiver);
		totalmesasge := ""
		for pi := 1; pi < len(params); pi++ {
			totalmesasge += params[pi] + " ";
		}
		channel.chan_msg(client, totalmesasge)
		return ;
	}
	for i := client.ListChain.Front(); i != nil; i = i.Next() {
		u := i.Value.(ClientChat)
		if *u.Name == receiver && u.LoggedIn {
			totalmesasge := ""
			for pi := 1; pi < len(params); pi++ {
				totalmesasge += params[pi] + " "
			}
			fmt.Printf("%s -> %s with message \"%s\"\n", *client.Name, *u.Name, totalmesasge)
			u.sendmsg(fmt.Sprintf("%s!~%s@", *client.Name, *client.Name), "PRIVMSG", *u.Name, totalmesasge);
			return
		}
	}
	client.sendmsg("", "401", *client.Name, params[0], ":No such nick/channel")
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
	client.sendmsg("", "353", *client.Name, "=", ch.Name, fmt.Sprintf(":%s", namereply))
	client.sendmsg("", "366", *client.Name, ch.Name, ":End of /NAMES list")
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
