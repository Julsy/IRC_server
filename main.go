package main

import (
	"fmt"
	"container/list"
	"strings"
	"net"
	"errors"
)

type ClientChat struct {
	Name	string
	In		chan string
	Out		chan string
	Conn 	net.Conn
	Quit	chan bool
	ListChain	*list.List
	ListChannel	*list.List
}

type ChannelChat struct {
	Name string
	Topic string
	UsersList *list.List
}

/*
** Still need to do these!
** Anything else we can add to this? This was off the top of my head lol
*/

var command_list = map[string]func(*ClientChat, []string) {
	"NICK": cmd_NICK,
	"USER": cmd_USER,
	"JOIN": cmd_JION,
	"PRIVMSG": cmd_PRIVMSG,
	"PING":	cmd_PING,
	"QUIT": cmd_QUIT,
	"WHO": cmd_WHO,
	"PART": cmd_PART,
	"LIST": cmd_LIST,
	"TOPIC": cmd_TOPIC,
}

/*
** We need to do sanitization of these params!! 
** We could probably generate another map and jump to the indexes based off of
** Error codes to make this more efficient!
**
** IE:
** Nick ->
** if len(params) == 0 {
**		client.sendmsg("", err["EMPTYNICK"], client.Nick, "Nick empty");
**		return ;
** }
** with the EMPTYNICK = 431 (ERR_NONICKNAMEGIVEN). (https://tools.ietf.org/html/rfc1459)
*/

func cmd_NICK(client *ClientChat, params []string) {
	client.Name = params[0];
	fmt.Printf("Nick to set %s (%s)", client.Name, params[0]);
}

func cmd_USER(client *ClientChat, params []string) {
	client.sendmsg("", 375, client.Name, "-:- Message of the day - ");
	client.sendmsg("", 372, client.Name, ":- We da cooliest");
	client.sendmsg("", 376, client.Name, ":End of /MOTD command");
}

func cmd_JOIN(client *ClientChat, params []string) {
	/*
	** Too Tired for this rn lmao - 
	** We need to essentially send the JOIN message (https://tools.ietf.org/html/rfc1459#section-4.2.1)
	** we then need to add user to the channel via first checking if the channel exists (channel_add())
	** Then adding the user to it via adduser();
	** We then need to send the connected client list as well as the channel TOPIC. 
	** We then need to send an updated connected client list to all clients currently connected 
	** IE: itterate through UsersLists and do send_list() with the addresss of current and the current channel
	*/
}

func client_recv(client *ClientChat) {
	buf := make([]byte, 0xFFFF);

	fmt.Println("client_recv(): start for: ", client.Name);
	for client.Read(buf) {
		for _, msg := range strings.Split(string(buf), "\r\n") {
			fmt.Println("GET: ", msg);
			command := strings.ToUpper(strings.Split(msg, " ")[0])
			params := strings.Split(msg, " ")[1:]
			cmd, found := command_list[command]
			if found {
				cmd(client, params);
			} else {
				fmt.Println("MSG: Unknown Message >>>", msg);
			}
		}
		// Clear it
		for i := 0; i < 0xFFFF; i++ {
			buf[i] = 0x0;
		}
	}
	fmt.Println("clientreceiver(): stop for: ", client.Name);
}

func send_list (clint *ClientChat, ch *ChannelChat) {
	namereply := "";
	for i := ch.UsersList.Front(); i != nil; i = i.Next() {
		c := i.Value.(ClientChat);
		namereply += cl.Name + " ";
	}
	client.sendmsg("", 353, client.Name, "=", client.Name, fmt.Sprintf(":%s", namereply));
	client.sendmsg("", 366, client.Name, ch.Name, " :End of /NAMES list");
}

func client_send(client *ClientChat) {
    fmt.Println("client_send(): start for: ", client.Name);
    for {
        fmt.Println("client_send(): wait for input to send");
        select {
            case buf := <- client.IN:
				fmt.Println("SEND:", buf)
                client.Conn.Write([]byte(buf + "\r\n"));
            case <-client.Quit:
                break;
        }
    }
    fmt.Println("client_send(): stop for: ", client.Name);
}

func clientHandle(conn net.Conn, userList *list.List, channellist *list.List) {
	newclient := &ClientChat {
		"",
		make(chan string),
		make(chan string),
		conn,
		make(chan bool),
		userList,
		channellist
	};
	go client_send(newclient);
	go client_recv(newclient);
	userList.PushBack(*newclient);
}

func main() {
	flag.Parse();
	fmt.Println("Starting IRC Server!");
	clientlist := list.New();
	channellist := list.New();
	listener, err := net.Listen("tcp", "0.0.0.0:1337");
	if err != nil {
		fmt.Println("Failed to bind: ", err);
		return ;
	}
	defer listener.Close();
	for {
		conn, err := listener.Accept();
		if err != nil {
			fmt.Println(err);
			return ;
		}
		go clientHandle(conn, clientlist, channellist);
	}
}