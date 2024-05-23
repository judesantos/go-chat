<template>
  <div class="container h-100">
    <div class="row justify-content-center h-100">
      <div class="col-12 form" v-if="!wsock">
        <div class="input-group">
          <input v-model="subscriber.subscribername" class="form-control subscribername" placeholder="subscribername"/>
          <input v-model="subscriber.password" type="password" class="form-control password" placeholder="password"/>

          <div class="input-group-append">
            <span class="input-group-text send_btn" @click="login">
              >
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
            v-model="channelInput"
            class="form-control name" 
            placeholder="Type the channel you want to join"
            @keyup.enter.exact="joinChannel"
            />
          <div class="input-group-append">
            <span class="input-group-text send_btn" @click="joinChannel">
              >
            </span>
          </div>
        </div>
      </div>

      <div class="chat" v-for="(channel, key) in channels" :key="key">
        <div class="card">
          <div class="card-header msg_head">
            <div class="d-flex bd-highlight justify-content-center">
              {{channel.name}}
              <span class="card-close" @click="leaveChannel(channel)">leave</span>
            </div>
          </div>
          <div class="card-body msg_card_body">
            <div 
            v-for="(message, key) in channel.messages"
              :key="key" 
              class="d-flex justify-content-start mb-4"
              >
              <div class="msg_cotainer">
                {{message.message}}
                <span class="msg_name" v-if="message.sender">{{message.sender.name}}</span>
              </div>
            </div>
          </div>
          <div class="card-footer">
            <div class="input-group">
              <textarea 
              v-model="channel.newMessage" 
              name=""
              class="form-control type_msg"
              placeholder="Type your message..."
                @keyup.enter.exact="sendMessage(channel)"
                ></textarea>
              <div class="input-group-append">
                <span class="input-group-text send_btn" @click="sendMessage(channel)">></span>
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

const	ACTION_SEND_MESSAGE          = "send-msg"
const	ACTION_JOIN_CHANNEL          = "join-channel"
const	ACTION_LEAVE_CHANNEL         = "leave-channel"
const	ACTION_LEFT_CHANNEL          = "left-channel"
const	ACTION_JOINED_CHANNEL        = "joined-channel"
//const	ACTION_NOTSUBSCRIBED_CHANNEL = "not-joined-channel"
const	ACTION_PRIVATE_CHANNEL       = "join-private-channel"

const WAIT_TIMEOUT_LIMIT = 16000
const SERVER_HOST = "http://localhost:8080"
const SOCKET_HOST = "ws://localhost:8080/ws"

let chatData = {
    channelInput: null,
    subscriber: {
      id: "",
      name: "",
      password: "",
      token: ""
    },
    channels: [],
    subscribers: [],

    wsock: null,

    loginError: "",
    waitTimeout: 0,
}

let chatOperations = {

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
        this.subscriber.token = result.data;
        this.wsConnect();
      }
    } catch (e) {
      this.loginError = "Login failed";
      console.error(e);
    }
  },

  wsConnect() {

    if (this.subscriber.token != "") {
      this.wsock = new WebSocket(SOCKET_HOST + "?jwt=" + this.subscriber.token);
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

      this.wsock.addEventListener('close', () => {

          this.wsock = null;
          this.reConnect()

      });
    }
  },

  reConnect() {
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

    for (let i = 0; i < data.length; i++) {

      let msg = JSON.parse(data[i]);

      switch (msg.requesttype) {

        case ACTION_SEND_MESSAGE:

          {
            const channel = this.findChannel(msg.target.id);
            if (typeof channel !== "undefined") {
              channel.messages.push(msg);
            }
          }

          break;

        case ACTION_JOIN_CHANNEL:

          {
            if(!this.subscriberFound(msg.sender)) {
              this.subscribers.push(msg.sender);
            }
          }

          break;

        case ACTION_LEFT_CHANNEL:

          {
            for (let i = 0; i < this.subscribers.length; i++) {
              if (this.subscribers[i].id == msg.sender.id) {
                this.subscribers.splice(i, 1);
                break;
              }
            }
          }

          break;

        case ACTION_JOINED_CHANNEL:

          {
            const channel = msg.target;
            channel.name = channel.private ? msg.sender.name : channel.name;
            channel["messages"] = [];

            this.channels.push(channel);
          }

          break;

        default:
          break;
      }

    }
  },

  sendMessage(channel) {

    if (channel.newMessage !== "") {
      this.wsock.send(JSON.stringify({
        action: ACTION_SEND_MESSAGE,
        message: channel.newMessage,
        target: {
          id: channel.id,
          name: channel.name
        }
      }));
      channel.newMessage = "";
    }
  },

  findChannel(channelId) {

    for (let i = 0; i < this.channels.length; i++) {
      if (this.channels[i].id === channelId) {
        return this.channels[i];
      }
    }
  },

  joinChannel() {
    this.wsock.send(JSON.stringify({
      action: ACTION_JOIN_CHANNEL,
      message: this.channelInput
    }));

    this.channelInput = "";
  },

  leaveChannel(channel) {

    this.wsock.send(JSON.stringify({
      action: ACTION_LEAVE_CHANNEL,
      message: channel.id
    }));

    for (let i = 0; i < this.channels.length; i++) {
      if (this.channels[i].id === channel.id) {
        this.channels.splice(i, 1);
        break;
      }
    }
  },

  joinPrivateChannel(channel) {

    this.wsock.send(JSON.stringify({
      action: ACTION_PRIVATE_CHANNEL,
      message: channel.id
    }));

  },

  subscriberFound(subscriber) {

    for (let i = 0; i < this.subscribers.length; i++) {
      if (this.subscribers[i].id == subscriber.id) {
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
  background-color: rgba(0, 0, 0, 0.4);
  border: 1px solid rgba(255, 255, 255, 0.2);
}

.card.profile {
  height:150px;
  margin: 15px;
}

.card.profile .card-header{
  color: #FFF;
}

.msg_head {
  color: #fff;
  text-align: center;
  font-size: 26px;
}

.msg_card_body {
  overflow-y: auto;
}
.card-header {
  border-radius: 15px 15px 0 0;
  border-bottom: 0;
}

.card-close {
  font-size: 0.5em;
  text-decoration: underline;
  float: right;
  position: absolute;
  right: 15px;
  cursor: pointer;
}

.card-footer {
  border-radius: 0 0 15px 15px;
  border-top: 0;
}
.container {
  align-content: center;
}

.type_msg {
  background-color: rgba(86, 10, 134, 0.6);
  border: 0;
  color: white;
  height: 60px;
  overflow-y: auto;
}
.type_msg:focus {
  box-shadow: none;
  outline: 0px;
  background-color: rgba(255, 255, 255, 0.6);
}

.send_btn {
  border-radius: 0 15px 15px 0;
  background-color: rgba(0, 0, 0, 0.3);
  border: 0;
  color: white;
  cursor: pointer;
}

.msg_cotainer {
  margin-top: auto;
  margin-bottom: auto;
  margin-left: 10px;
  border-radius: 25px;
  background-color: #82ccdd;
  padding: 10px 15px;
  position: relative;
  min-width: 100px;
  line-height: 1.2rem;
}
.msg_cotainer_send {
  margin-top: auto;
  margin-bottom: auto;
  margin-right: 10px;
  border-radius: 25px;
  background-color: #75d5fd;
  padding: 10px;
  position: relative;
}

.msg_name {
  display: block;
  font-size: 0.8em;
  font-style: italic;
  color: #545454;
}

.msg_head {
  position: relative;
}

</style>
