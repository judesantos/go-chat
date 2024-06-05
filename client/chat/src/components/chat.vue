<template>
  <div class="chat-container">
    <div class="chat-welcome" v-if="!channel.messages">
      <h1>
        Let's go-chat! 
      </h1>
    </div>
    <div class="chat" v-if="channel.messages">
      <div class="chat-header">
        <h2> # {{ channel.name }}</h2>
      </div>
      <div class="chat-messages">
        <div class="message" v-for="(message, index) in channel.messages" :key="index">
          <span class="username">{{ message.sender }}:</span>
          <span class="text">{{ message.message }}</span>
        </div>
      </div>
      <div class="chat-input">
        <input type="text" v-model="newMessage" @keyup.enter="sendMessage" placeholder="Type your message...">
        <button @click="sendMessage">Send</button>
      </div>
    </div>
  </div>
</template>

<script>
export default {
  name: 'ChatComponent',
  props: {
    channel: Object,
    messages: Array
  },
  data() {
    return {
      newMessage: ''
    }
  },
  methods: {
    sendMessage() {
      let newMessage = this.newMessage.trim()
      if (newMessage.length) {
        this.$emit('sendMessage', {message: newMessage})
      }
      this.newMessage = ''
    }
  }
}
</script>

<style>
.chat {
  width: 100%;
  display: flex;
  flex-direction: column;
  border-left: 1px solid rgba(193, 192, 192, 0.274); 
}

.chat-container {
  display: flex;
  width: 80%;
  margin: 0;
  max-width: none;
}

.chat-welcome{
  width: 100%;
  height: 100%;
  display: flex;
  align-items: center; /* Vertical centering */
}

.chat-welcome h1 {
  width: 100%;
  text-align: center;
  color:rgba(25, 217, 70, 0.879);
  font-weight: bolder;
  font-size: 5.5em;
}

.chat-header {
  /*
  background-color: #3498db;
  color: rgba(16, 16, 16, 0.686);
  */
  background-color: rgba(143, 37, 219, 0.979);
  color: white;
  padding: 20px;
}

.chat-messages {
  flex: 1;
  padding: 20px;
  overflow-y: auto;
}

.message {
  margin-bottom: 10px;
}

.username {
  font-weight: bold;
  margin-right: 10px;
}

.chat-input {
  display: flex;
  padding: 10px;
  border-top: 1px solid #ddd;
}

.chat-input input {
  flex: 1;
  padding: 10px;
  border: 1px solid #ddd;
  border-radius: 4px;
}

.chat-input button {
  padding: 10px 20px;
  border: none;
  background-color: rgba(52, 24, 190, 0.895);
  color: white;
  cursor: pointer;
  margin-left: 10px;
  border-radius: 5px;
}

.chat-input button:hover {
  background-color: rgba(52, 24, 190, 0.803);
}
</style>
