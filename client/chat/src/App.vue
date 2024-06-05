<template>
  <div class="container">
    <div class="login-container" v-if="!wsock">
      <div class="login-form">
        <h2 class="text-center">Login</h2>
        <div class="form-group">
            <label for="username">Username</label>
            <input
              type="text"
              class="form-control" 
              placeholder="Enter username"
              v-model="subscriber.name" 
            />
        </div>
        <div class="form-group">
            <label for="password">Password</label>
            <input 
              type="password"
              class="form-control"
              placeholder="Enter password" 
              v-model="subscriber.password" 
              @keyup.enter.exact="login"/>
        </div>
        <div class="form-group submit">
          <button type="submit" class="btn btn-primary btn-block"
            @click="login"
          >
            Login
          </button>
        </div>
      </div>
    </div>

    <Sidebar v-if="wsock" 
      :channels="channels" 
      @channelSelected="selectChannel" 
      @channelRemoved="leaveAndRemoveChannel"
      @newChannel="newChannel"
      @logout="logout"
    />

    <Chat v-if="wsock" :channel="selectedChannel" 
      @sendMessage="addMessage"
    />
  </div>
</template>

<script>

  import Sidebar from './components/Sidebar.vue'
  import Chat from './components/Chat.vue'

  import axios from 'axios';
  const { v1: uuidv1 } = require('uuid');

  const REQ_SEND_MESSAGE = "message";
  const REQ_JOIN_CHANNEL = "join-channel";
  const REQ_LEAVE_CHANNEL = "leave-channel";
  const REQ_JOINED_CHANNEL = "joined-channel";
  const REQ_SUBSCRIBER_JOINED = "subscriber-joined";
  const REQ_SUBSCRIBER_LEFT   = "subscriber-left";
  const REQ_JOIN_PRIVATE_CHANNEL = "join-private-channel";

  const MESSAGE_TYPE = {
    REQ: 0,
    ACK: 1,
    BROADCAST: 2
  }

  const WAIT_TIMEOUT_LIMIT = 16000
  const SERVER_HOST = "http://localhost:8080"
  const SOCKET_HOST = "ws://localhost:8080/ws"

  const channels = []
  const chatData = {
      selectedChannel: {},
      channelName: null,
      subscriber: {
        id: "",
        name: "",
        password: "",
        email: "jude.msantos@gmail.com",
        token: ""
      },
      channels,
      subscribers: [],

      wsock: null,
      loggedOut: false,

      loginError: "",
      waitTimeout: 0,
  }

  const chatOperations = {

    selectChannel(channel) {
      this.selectedChannel = channel;
    },

    addMessage(objMsg) {
      this.selectedChannel.newMessage = objMsg.message
      this.selectedChannel.messages.push({
        sender: this.subscriber.name,
        message: objMsg.message
      })
      this.doSendMessage(this.selectedChannel)
    },

    newChannel(channel) {
      //this.channels.push(channel)
      this.channelName = channel.name
      this.joinChannel()
    },

    leaveAndRemoveChannel(channel) {
      this.leaveChannel(channel)
      if (this.channels.length) {
        this.selectedChannel = this.channels[this.channels.length - 1]
      } else {
        this.selectedChannel = {}
      }
    },

    async logout() {

      for (let channel of this.channels) {
        this.leaveChannel(channel)
      }

      this.selectedChannel = {}
      this.channels = []

      this.subscriber.name = ''
      this.subscriber.password = ''

      if (this.wsock) {

        this.loggedOut = true

        this.wsock.close()
        this.wsock = null

      }
    },

    async login() {

      this.loggedOut = false;

      try {

        const result = await axios.post(SERVER_HOST + "/login", this.subscriber);

        if (
          result.data.status !== "undefined" && 
          result.data.status == "error"
        ) {

          console.error("Error: " + result)
          this.loginError = "Login failed";

        } else {

          console.log("Login Success: " + JSON.stringify(result.data))
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

      if (this.loggedOut)
        return

      console.log("RECONNECT")

      if (this.waitTimeout < WAIT_TIMEOUT_LIMIT) {
        this.waitTimeout *= 2;
        this.wsConnect();
      } else {
        console.error("Reconnect failed: Connection wait timed out.")
        this.loginError = "Connection Lost!"
      }
    },

    wsMessage(event) {

      let data = event.data;
      data = data.split(/\r?\n/);

      console.log("received message: " + data)

      for (let i = 0; i < data.length; i++) {

        let msg = JSON.parse(data[i]);

        //if (msg.messagetype != "1")
          // non-ACK response?
          //continue;

        switch (msg.requesttype) {

          case REQ_SEND_MESSAGE:

            {
              let channel = this.findChannel(msg.channelname);
              if (channel) {
                let sender = msg.session.subscriber.name
                if (msg.session.subscriber.name === this.subscriber.name) {
                  sender = 'you'
                }
                if (msg.messagetype === MESSAGE_TYPE.BROADCAST) {
                  if (msg.requestsubtype === REQ_JOINED_CHANNEL) {
                    sender = msg.channelname
                  }
                  if (msg.session.subscriber.name !== this.subscriber.name) {
                    channel.messages.push({
                      sender,
                      message: msg.message
                    });
                  }
                } else if (msg.messagetype !== MESSAGE_TYPE.ACK) {
                  channel.messages.push({
                    sender,
                    message: msg.message
                  });

                }
              }
            }

            break;

          case REQ_SUBSCRIBER_JOINED:

            {
                if(!this.subscriberFound(msg.subscriber)) {
                  this.subscribers.push(msg.subscriber);
                }
            }

            break;

          case REQ_SUBSCRIBER_LEFT:

            {
              for (let i = 0; i < this.subscribers.length; i++) {
                if (this.subscribers[i].name == msg.subscriber.name) {
                  this.subscribers.splice(i, 1);
                  break;
                }
              }
            }

            break;

          case REQ_JOINED_CHANNEL:

            {
              if (!this.findChannel(msg.channelname)) {
                const channel = {
                  name: msg.channelname,
                  messages: []
                }
                let sender = msg.session.subscriber.name 
                if (msg.messagetype != MESSAGE_TYPE.REQ) {
                  sender = msg.channelname
                }
                channel.messages.push({
                  sender,
                  message: msg.message
                })
                this.channels.push(channel);
                this.selectChannel(channel)
              }
            }

            break;

          default:
            break;
        }

      }
    },

    doSendMessage(channel) {
      this.sendMessage(channel.name, REQ_SEND_MESSAGE, channel.newMessage)
      channel.newMessage = ''
    },



    newMessage(reqType, channelName, message = '') {
      return JSON.stringify({
        id: uuidv1(),
        messagetype: 0, // Req.
        requesttype: reqType,
        message,   
        channelname: channelName,    
        subscriber: {
          name: this.subscriber.name,
          email: this.subscriber.email
        }
      })
    },

    sendMessage(channelName, reqType, msg = '') {

      if (!this.wsock) {
        this.loginError = "Connection lost. Please sign-in."
        return
      }

      let message = this.newMessage(
        reqType,
        channelName,
        msg
      )
      console.log("Send message: " + message)
      this.wsock.send(message);
    },

    findChannel(channelName) {

      for (let i = 0; i < this.channels.length; i++) {
        if (this.channels[i].name === channelName) {
          return this.channels[i];
        }
      }
      return null
    },

    joinChannel() {

      this.sendMessage(this.channelName, REQ_JOIN_CHANNEL, "Hello " + this.channelName)
      this.channelName = "";
    },

    leaveChannel(channel) {

      this.sendMessage(channel.name, REQ_LEAVE_CHANNEL)

      for (let i = 0; i < this.channels.length; i++) {
        if (this.channels[i].name === channel.name) {
          this.channels.splice(i, 1);
          break;
        }
      }
    },

    joinPrivateChannel(channel) {

      console.log("joinPrivateChannel")
      this.sendMessage(channel.name, REQ_JOIN_PRIVATE_CHANNEL, channel.name)
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
    name: 'App',
    components: {
      Sidebar,
      Chat
    },
    data() {
      return chatData
    },
    methods: chatOperations
  }

</script>

<style>

body,
html {
  height: 100%;
  margin: 0;
  /*
  background: #1018a47e;
  background: -webkit-linear-gradient(to right, #560a86, #7122fa, #560a86);
  background: linear-gradient(to right, #560a86, #7122fa, #560a86);
  */
}

* {
  margin: 0;
  padding: 0;
  box-sizing: border-box;
}

body {
  font-family: Arial, sans-serif;
  display: flex;
  height: 100vh;
}

.container {
  display: flex;
  width: 100%;
  margin: 0;
  max-width: none;
}

.login-container {
  display: flex;
  justify-content: center;
  align-items: center;
  width: 100%; 
  height: 100vh;
}

.login-form {
  width: 100%;
  max-width: 400px;
  padding: 15px;
  border: 1px solid #ddd;
  border-radius: 5px;
  background-color: #f7f7f7;
}

.form-group {
  margin: 10px;
}

.form-group label {
  margin: 5px 0;
}

.submit {
  padding-top: 5px;
  display: flex;
  justify-content: end;
}

</style>
