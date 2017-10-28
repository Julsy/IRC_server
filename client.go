package main

import (
	"fmt"
)

func (c *ClientChat) sendmsg(prefix string, command string, params ...string) {
	var msg string;
	msg = fmt.Sprintf(": %s %s", prefix, command);
	for _, v := range params {
		msg = msg + " " + v;
	}
	c.IN <- msg;
}

func (c *ClientChat) Close() {
	c.Quit <- true;
	c.Conn.Close();
	c.delete();
	fmt.Println(c.Name, " Quit");
	fmt.Println("ClientList: %d", c.ListChain.Len())
}

func (c *ClientChat) delete() {
	for i := c.ListChain.Front(); i != nil; i = i.Next() {
		client := i.Value.(ClientChat);
		if c.Conn == client.Conn {
			c.ListChain.Remove(i);
		}
	}
	for i := c.ListChain.Front(); i != nil; i = i.Next() {
		channel := i.Value.(ChannelChat);
		channel.removeuser(c);
	}
}

func (c *ClientChat) sendmsg(prefix string, command string, params ...string) {
	var msg string;

	msg = fmt.Sprintf(":%s %s", prefix, command);
	for _, v := range params {
		msg = msg + " " + v;
	}
	c.IN <- msg;
}

func (c *ClientChat) channel_add(name string) *ChannelChat {
	for i := c.ListChannel.Front(); i != nil; i = i.Next() {
		ch := i.Value.(ChannelChat);
		if ch.Name == name {
			fmt.Println("Found");
			return &ch;
		}
	}
	ch := &ChannelChat{
		Name:	name,
		UserList:	list.New()
	};
	c.ListChannel.PushBack(*ch);
	fmt.Println("Created channel ", name);
}

func (ch *ChannelChat) adduser(user *ClientChat) {
	ch.UserList.PushBack(*user);
}

func (ch *ChannelChat) deluser(user *ClientChat) {
	for i := ch.UserList.Front(); i != nil; i = i.Next() {
		client := i.Value.(ClientChat);
		if user.Conn == client.Conn {
			ch.UserList.Remove(i);
			ch.updatelist();
		}
	}
}

func (ch *ChannelChat) updatelist() {
	for i := i.UserList.Front(); i != nil; i = i.Next() {
		client := cl.Value.(ClientChat);
		send_list(&client, ch);
	}
}