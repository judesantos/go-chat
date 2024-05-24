<template>
  <div class="container h-100">
    <div class="row justify-content-center h-100">
      <div class="col-12 form" v-if="!wsock">
        <div class="input-group">
          <input v-model="subscriber.name" class="form-control subscribername" placeholder="subscribername"/>
          <input v-model="subscriber.password" type="password" class="form-control password" placeholder="password"/>

          <div class="input-group-append">
            <span class="input-group-text submit-button" @click="login">
              Sign-in
            </span>
          </div>
        </div>

        <div class="alert alert-danger" role="alert" v-show="loginError">
          {{loginError}}
        </div>

      </div>
      <div class="col-12 ">
        <div class="row">
          <div class="col-2 card profile" v-for="subscriber in subscribers" :key="subscriber.id">
            <div class="card-header">{{subscriber.name}}</div>
            <div class="card-body">
              <button class="btn btn-primary" @click="joinPrivateChannel(subscriber)">Send Message</button>
            </div>
          </div>
        </div>
      </div>
      <div class="col-12 channel" v-if="wsock != null">
        <div class="input-group">
          <input
            v-model="channelName"
            class="form-control name" 
            placeholder="Type the channel you want to join"
            @keyup.enter.exact="joinChannel"
            />
          <div class="input-group-append">
            <span class="input-group-text submit-button" @click="joinChannel">
              Join Channel
            </span>
          </div>
        </div>
      </div>

      <div class="chat" v-for="(channel, key) in channels" :key="key">
        <div class="card">
          <div class="card-header message-head">
            <div class="d-flex bd-highlight">
              {{channel.name}}
              <span class="close_icon" @click="leaveChannel(channel)">&#10005;</span>
            </div>
          </div>
          <div class="card-body card-body">
            <div 
            v-for="(message, key) in channel.messages"
              :key="key" 
              class="d-flex justify-content-start mb-1"
              >
              <div class="message-group">
                <span class="message-name" v-if="message.sender">{{message.sender.name}}:</span>
                  {{message.message}}
              </div>
            </div>
          </div>
          <div class="card-footer">
            <div class="input-group">
              <textarea 
              v-model="channel.newMessage" 
              name=""
              class="form-control input-message"
              placeholder="Type your message..."
                @keyup.enter.exact="doSendMessage(channel)"
                ></textarea>
              <div class="input-group-append">
                <span class="input-group-text submit-button" @click="doSendMessage(channel)">
                  Send
                </span>
              </div>
            </div>
          </div>
        </div>
      </div>
    </div>
  </div>
</template>

<script>
import axios from 'axios';
const { v1: uuidv1 } = require('uuid');


const	ACTION_MESSAGE         = "send-msg"
const	ACTION_JOIN_CHANNEL    = "join-channel"
const	ACTION_LEAVE_CHANNEL   = "leave-channel"
const	ACTION_LEFT_CHANNEL    = "left-channel"
const	ACTION_JOINED_CHANNEL  = "joined-channel"
const	ACTION_PRIVATE_CHANNEL = "join-private-channel"

//const MESSAGE_TYPE = {
//  REQ: 0,
//  ACK: 1,
//  BROADCAST: 2
//}

const WAIT_TIMEOUT_LIMIT = 16000
const SERVER_HOST = "http://localhost:8080"
const SOCKET_HOST = "ws://localhost:8080/ws"

const chatData = {
    channelName: null,
    subscriber: {
      id: "",
      name: "",
      password: "",
      email: "jude.msantos@gmail.com",
      token: ""
    },
    channels: [],
    subscribers: [],

    wsock: null,

    loginError: "",
    waitTimeout: 0,
}

const chatOperations = {

  async login() {

    try {

      const result = await axios.post(SERVER_HOST + "/login", this.subscriber);

      if (
        result.data.status !== "undefined" && 
        result.data.status == "error"
      ) {

        console.error("Error: " + result)
        this.loginError = "Login failed";

      } else {

        const response = result.data
        if (response.status == 'success') {

          this.subscriber.name = response.name
          this.subscriber.email = response.email
          this.subscriber.token = response.token

          this.wsConnect();

        } else {

          console.log("Login failed: " + response)
          return

        }

      }
    } catch (e) {
      this.loginError = "Login failed";
      console.error(e);
    }
  },

  wsConnect() {

    if (this.subscriber.token != "") {
      this.wsock = new WebSocket(SOCKET_HOST + "?jwt=" + this.subscriber.token.AccessToken);
    } else if (this.subscriber.name != "") {
      this.wsock = new WebSocket(SOCKET_HOST + "?name=" + this.subscriber.name);
    }

    if (this.wsock) {

      this.wsock.addEventListener('error', (e) => { 
        console.error(e)
        this.wsock = null;
      });

      this.wsock.addEventListener('open', () => {
        console.log("connected to chat server!");
        this.waitTimeout = 1000;
      });

      this.wsock.addEventListener('message', (event) => { 
        this.wsMessage(event) 
      });

      this.wsock.addEventListener('close', (e) => {

        console.log("Console closed: " + JSON.stringify(e))

        this.wsock = null;
        this.reConnect()

      });
    }
  },

  reConnect() {

    console.log("RECONNECT")

    if (this.waitTimeout < WAIT_TIMEOUT_LIMIT) {
      this.waitTimeout *= 2;
      this.wsConnect();
    } else {
      console.error("Reconnect failed: Connection wait timed out.")
      this.reConnect()
    }
  },

  wsMessage(event) {

    let data = event.data;
    data = data.split(/\r?\n/);

    console.log("wsMessage: " + data)

    for (let i = 0; i < data.length; i++) {

      let msg = JSON.parse(data[i]);

      if (msg.messagetype != "1")
        // non-ACK response?
        continue;

      switch (msg.requesttype) {

        case ACTION_MESSAGE:

          console.log("ACTION_MESSAGE response: " + data[i])
          {
            const channel = this.findChannel(msg.channelname);
            if (typeof channel !== "undefined") {
              channel.messages.push(msg);
            }
          }

          break;

        case ACTION_JOIN_CHANNEL:

          console.log("ACTION_JOIN_CHANNEL response: " + data[i])
          {
              if(!this.subscriberFound(msg.subscriber)) {
                this.subscribers.push(msg.subscriber);
              }
          }

          break;

        case ACTION_LEFT_CHANNEL:

          console.log("ACTION_LEFT_CHANNEL response: " + data[i])
          {
            for (let i = 0; i < this.subscribers.length; i++) {
              if (this.subscribers[i].name == msg.subscriber.name) {
                this.subscribers.splice(i, 1);
                break;
              }
            }
          }

          break;

        case ACTION_JOINED_CHANNEL:

          console.log("ACTION_JOINED_CHANNEL response: " + data[i])
          {
            const channel = {
              name: msg.channelname,
              messages: []
            }
            this.channels.push(channel);
          }

          break;

        default:
          break;
      }

    }
  },

  doSendMessage(channel) {
    this.sendMessage(channel)
    channel.messages.push({
      message: channel.newMessage,
      sender: {
        name: this.subscriber.name
      }
    })
    channel.newMessage = ""
  },

  newMessage(reqType, channelName, message = '') {
    return JSON.stringify({
      id: uuidv1(),
      messagetype: 0, // Req.
      requesttype: reqType,
      message: message,   
      channelname: channelName,    
      subscriber: {
        name: this.subscriber.name,
        email: this.subscriber.email
      }
    })
  },

  sendMessage(channel) {

    if (channel.newMessage !== "") {

      let message = this.newMessage(
        ACTION_MESSAGE,
        channel.name,
        channel.newMessage
      )
      this.wsock.send(message);
    }
  },

  findChannel(channelId) {

    for (let i = 0; i < this.channels.length; i++) {
      if (this.channels[i].id === channelId) {
        return this.channels[i];
      }
    }
    return null
  },

  joinChannel() {

    const message = this.newMessage(
      ACTION_JOIN_CHANNEL,
      this.channelName,
      "Hello " + this.channelName
    )
    console.log("joinChannel send message: " + message)
    let channel = this.findChannel(this.channelName);
    if (channel == null) {
      channel = {
        name: this.channelName,
        messages: []
      };
      console.log("add channel: " + this.channelname)
      this.channels.push(channel);
    }

    this.wsock.send(message);
    this.channelName = "";
  },

  leaveChannel(channel) {

    const message = this.newMessage(
      ACTION_LEAVE_CHANNEL,
      channel.name
    )
    console.log("leaveChannel send message: " + message)
    this.wsock.send(message);

    for (let i = 0; i < this.channels.length; i++) {
      if (this.channels[i].id === channel.id) {
        this.channels.splice(i, 1);
        break;
      }
    }
  },

  joinPrivateChannel(channel) {

    console.log("joinPrivateChannel Send message: " + message)
    const message = this.newMessage(
      ACTION_PRIVATE_CHANNEL,
      channel.name
    )
    this.wsock.send(message);
  },
  
  subscriberFound(subscriber) {

    for (let i = 0; i < this.subscribers.length; i++) {
      if (this.subscribers[i].name == subscriber.name) {
        return true;
      }
    }

    return false;
  } 

}

export default {
  name: 'ChatComponent',
  data() {
    return chatData
  },
  methods: chatOperations
}

</script>

<!-- Add "scoped" attribute to limit CSS to this component only -->
<style scoped>
body,
html {
  height: 100%;
  margin: 0;
  background: #0310ea;
  background: -webkit-linear-gradient(to right, #560a86, #7122fa, #560a86);
  background: linear-gradient(to right, #560a86, #7122fa, #560a86);
}

#app {
  height: 100%;
}

.chat {
  margin: 15px;
}

.channel,
.form {
  margin-top: auto;
  margin-bottom: auto;
}

.card {
  height: 500px;
  border-radius: 15px;
  background-color: rgba(146, 145, 145, 0.02);
  border: 1px solid rgba(242, 229, 229, 0.7);
}

.card.profile {
  height:150px;
  margin: 15px;
}

.card.profile .card-header{
  color: #FFF;
}

.message-head {
  font-size: 26px;
}

.card-body {
  overflow-y: auto;
}

.card-header {
  border-radius: 15px 15px 0 0;
  border-bottom: 0;
}

.card-close {
  font-size: 0.5em;
  float: right;
  position: absolute;
  right: 15px;
  cursor: pointer;
}

.card-footer {
  border-radius: 0 0 15px 15px;
  border-top: 1px solid rgba(242, 229, 229, 0.7);
  background-color: rgb(230 229 229 / 2%);
}

.container {
  align-content: center;
}

.input-message {
  background-color: rgb(81, 79, 79, 0.03);
  color: rgb(60, 60, 60);
  overflow-y: auto;
}
.input-message:focus {
  box-shadow: none;
  outline: 0px;
  background-color: rgb(255, 255, 255);
}

.submit-button {
  border-radius: 5px;
  margin: 0px 5px 0px 5px;
  background-color: rgba(52, 24, 190, 0.895);
  border: 0;
  color: white;
  cursor: pointer;
}

.message-group {
  color: rgb(60, 60, 60);
  text-align: left;
  margin-left: 5px;
  border-radius: 2px;
  padding: 2px 5px;
  position: relative;
  min-width: 100px;
  line-height: 1.2rem;
}
.message-group-send {
  margin-top: auto;
  margin-bottom: auto;
  margin-right: 10px;
  border-radius: 25px;
  background-color: #75d5fd;
  padding: 10px;
}

.message-name {
  font-size: 1em;
  font-style: italic;
  color: #0000005c;
}

.message-head {
  position: relative;
}

.close_icon {
  position: absolute;
  right: 15px;
  float: right;
  font-size: 16px;
  font-weight: 600;
  cursor: pointer;
  color: rgb(176 163 163 / 73%);
}

</style>
