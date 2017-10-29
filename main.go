package main

import (
	"fmt"
	"container/list"
	"strings"
	"net"
	"flag"
	// "errors"
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
	"JOIN": cmd_JOIN,
	// "PRIVMSG": cmd_PRIVMSG,
	// "PING":	cmd_PING,
	// "QUIT": cmd_QUIT,
	// "WHO": cmd_WHO,
	// "PART": cmd_PART,
	// "LIST": cmd_LIST,
	// "TOPIC": cmd_TOPIC,
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
	// for i := client.ListChain.Front(); i != nil; i = i.Next() {
	// 	if i.Value.(ClientChat).Name == params[0] {
	// 		client.sendmsg("", "433", client.Name, ":Nickname is already in use");
	// 		fmt.Printf("Nick is already taken: %s ", params[0]);
	// 		return ;
	// 	}
	// }
	// if (client.Name != "") {
	// 	fmt.Printf("Setting Nick (%s) to %s\n", client.Name, params[0])
	// }
	fmt.Printf("In nickset(%p)\n", client);
	client.Name = params[0];
	fmt.Printf("Nick to set %s (%s)\n", client.Name, params[0]);
}

func cmd_USER(client *ClientChat, params []string) {
	client.sendmsg("", "375", client.Name, "-:- Message of the day - ");
	client.sendmsg("", "372", client.Name, ":- We da cooliest");
	client.sendmsg("", "376", client.Name, ":End of /MOTD command");
}

func cmd_JOIN(client *ClientChat, params []string) {
    channel := client.channel_add(params[0]);
    channel.adduser(client);
    client.sendmsg("", 332, ch.Name, " :", ch.Topic);
    send_list(client, channel);
    for i := channel.UsersLists.Front(); i != nil; i = i.Next() {
        c := i.Value.(ClientChat);
        send_list(c, channel);
    }
}

func (c *ClientChat) nick_open(nickname string) (bool) {
	for i := c.ListChain.Front(); i != nil; i = i.Next() {
		if i.Value.(ClientChat).Name == nickname {
			return true;
		}
	}
	return false;
}

func list_all(client *ClientChat) {
	fmt.Println("=================== Listing all Users ===============")
	for i := client.ListChain.Front(); i != nil; i = i.Next() {
		c := i.Value.(ClientChat);
		fmt.Printf("(%p) %s - %s\n", &c, c.Conn.LocalAddr().String(), c.Name);
	}
	fmt.Println("=====================================================");
}

func client_recv(client *ClientChat) {
	buf := make([]byte, 0xFFFF);

	fmt.Println("client_recv(): start for: ", client.Name);
	for {
        _, err := client.Conn.Read(buf);
        if err != nil {
            fmt.Println("Err", err);
            break ;
        }
        for _, msg := range strings.Split(string(buf), "\r\n") {
            fmt.Println("GET: ", msg);
            command := strings.ToUpper(strings.Split(msg, " ")[0]);
            params := strings.Split(msg, " ")[1:]
            cmd, found := command_list[command]
            if found {
                cmd(client, params);
				list_all(client);
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

func send_list (client *ClientChat, ch *ChannelChat) {
	namereply := "";
	for i := ch.UsersList.Front(); i != nil; i = i.Next() {
		c := i.Value.(ClientChat);
		namereply += c.Name + " ";
	}
	client.sendmsg("", "353", client.Name, "=", client.Name, fmt.Sprintf(":%s", namereply));
	client.sendmsg("", "366", client.Name, ch.Name, " :End of /NAMES list");
}

func client_send(client *ClientChat) {
	for {
		select {
			case buf := <- client.In :
				client.Conn.Write([]byte(buf + "\r\n"));
			case <-client.Quit:
				break;
		}
	}
	fmt.Printf("client_send() exited.\n");
}

func clientHandle(conn net.Conn, userList *list.List, channellist *list.List) {
	newclient := &ClientChat {
		"Test",
		make(chan string),
		make(chan string),
		conn,
		make(chan bool),
		userList,
		channellist,
	};
	userList.PushBack(*newclient);
	c := userList.Back().Value.(ClientChat);
	// fmt.Printf("newclient = %p userList.Back() = %p\n", newclient, userList.Back().Value)
	go client_send(&c);
	go client_recv(&c);
	list_all(newclient);
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
