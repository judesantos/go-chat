<template>
  <div class="sidebar">
    <h2>Channels<span class="new-channel" @click="newChannel">+</span></h2>
    <ul>
      <li v-for="channel in channels" :key="channel.name" @click="selectChannel(channel)">
        # {{ channel.name }} <span class="leave-channel" @click="leaveChannel(channel)">&#10005;</span>
      </li>
    </ul>
  </div>
</template>

<script>
export default {
  name: 'SidebarComponent',
  props: {
    channels: []
  },
  methods: {
    selectChannel(channel) {
      this.$emit('channelSelected', channel)
    },
    leaveChannel(channel) {
      if (confirm("Remove Channel " + channel.name + "?")) {
        this.$emit("channelRemoved", channel)
      }
    },
    newChannel() {
      const newChannelName = prompt("Enter new channel name.")
      if (newChannelName.length)
        this.$emit("newChannel", {name: newChannelName, messages: []})
    }
  }
}
</script>

<style>
.sidebar {
  width: 20%;
  background-color: #ababac45;
  color: rgba(16, 16, 16, 0.686);
  padding: 20px;
}

.sidebar h2 {
  margin-bottom: 20px;
  display: flex;
  justify-content: space-between;
}

.sidebar h2:hover .new-channel{
  opacity: 0.75;
  transition: ease-out 0.21s;
}

.sidebar ul {
  list-style:none;
  padding-left: 0;
}

.sidebar ul li {
  padding: 5px 14px 5px 5px;
  cursor: pointer;
  display: flex;
  justify-content: space-between;
}

.sidebar ul li:hover {
  background-color: #797b7c37;
}

.sidebar ul li:hover .leave-channel {
  opacity: 0.75;
  transition: ease-out 0.21s;
}

.new-channel, .leave-channel{
  cursor: pointer;
  opacity: 0;
  font-weight:lighter;
  color: rgba(187, 182, 182, 0.925)
}

.new-channel:hover, .leave-channel:hover {
  color: gray
}

</style>
