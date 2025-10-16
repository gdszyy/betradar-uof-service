/**
 * Betradar UOF 前端客户端
 * 连接到Go后端服务的WebSocket
 */

(function(window) {
  'use strict';

  class UOFClient {
    constructor(config = {}) {
      this.config = {
        wsUrl: config.wsUrl || this.getDefaultWSUrl(),
        apiUrl: config.apiUrl || this.getDefaultAPIUrl(),
        autoReconnect: config.autoReconnect !== false,
        reconnectInterval: config.reconnectInterval || 3000,
        ...config
      };

      this.ws = null;
      this.isConnected = false;
      this.reconnectTimer = null;
      this.eventHandlers = {};
      this.messageFilters = [];
      this.eventFilters = [];
    }

    getDefaultWSUrl() {
      const protocol = window.location.protocol === 'https:' ? 'wss:' : 'ws:';
      return `${protocol}//${window.location.host}/ws`;
    }

    getDefaultAPIUrl() {
      return `${window.location.protocol}//${window.location.host}/api`;
    }

    // 连接到WebSocket
    connect() {
      if (this.ws) {
        console.warn('Already connected');
        return;
      }

      console.log('Connecting to', this.config.wsUrl);

      try {
        this.ws = new WebSocket(this.config.wsUrl);

        this.ws.onopen = () => {
          this.isConnected = true;
          console.log('Connected to UOF WebSocket');
          this.emit('connected');

          // 发送订阅消息
          if (this.messageFilters.length > 0 || this.eventFilters.length > 0) {
            this.subscribe(this.messageFilters, this.eventFilters);
          }
        };

        this.ws.onmessage = (event) => {
          this.handleMessage(event.data);
        };

        this.ws.onerror = (error) => {
          console.error('WebSocket error:', error);
          this.emit('error', error);
        };

        this.ws.onclose = () => {
          this.isConnected = false;
          this.ws = null;
          console.log('Disconnected from UOF WebSocket');
          this.emit('disconnected');

          // 自动重连
          if (this.config.autoReconnect) {
            this.reconnectTimer = setTimeout(() => {
              console.log('Reconnecting...');
              this.connect();
            }, this.config.reconnectInterval);
          }
        };

      } catch (error) {
        console.error('Failed to connect:', error);
        this.emit('error', error);
      }
    }

    // 断开连接
    disconnect() {
      this.config.autoReconnect = false;

      if (this.reconnectTimer) {
        clearTimeout(this.reconnectTimer);
        this.reconnectTimer = null;
      }

      if (this.ws) {
        this.ws.close();
        this.ws = null;
      }
    }

    // 订阅消息
    subscribe(messageTypes = [], eventIds = []) {
      this.messageFilters = messageTypes;
      this.eventFilters = eventIds;

      if (this.isConnected) {
        this.send({
          type: 'subscribe',
          message_types: messageTypes,
          event_ids: eventIds
        });
      }
    }

    // 取消订阅
    unsubscribe() {
      this.messageFilters = [];
      this.eventFilters = [];

      if (this.isConnected) {
        this.send({
          type: 'unsubscribe'
        });
      }
    }

    // 发送消息
    send(data) {
      if (!this.isConnected) {
        console.warn('Not connected');
        return;
      }

      this.ws.send(JSON.stringify(data));
    }

    // 处理接收到的消息
    handleMessage(data) {
      try {
        const message = JSON.parse(data);

        // 触发通用消息事件
        this.emit('message', message);

        // 触发特定类型的事件
        if (message.type) {
          this.emit(message.type, message);
        }

        if (message.message_type) {
          this.emit(message.message_type, message);
        }

      } catch (error) {
        console.error('Failed to parse message:', error);
      }
    }

    // 事件监听
    on(event, handler) {
      if (!this.eventHandlers[event]) {
        this.eventHandlers[event] = [];
      }
      this.eventHandlers[event].push(handler);
    }

    // 移除事件监听
    off(event, handler) {
      if (!this.eventHandlers[event]) return;

      if (handler) {
        this.eventHandlers[event] = this.eventHandlers[event].filter(h => h !== handler);
      } else {
        delete this.eventHandlers[event];
      }
    }

    // 触发事件
    emit(event, ...args) {
      if (!this.eventHandlers[event]) return;

      this.eventHandlers[event].forEach(handler => {
        try {
          handler(...args);
        } catch (error) {
          console.error('Event handler error:', error);
        }
      });
    }

    // API方法

    // 获取消息列表
    async getMessages(params = {}) {
      const query = new URLSearchParams(params).toString();
      const url = `${this.config.apiUrl}/messages${query ? '?' + query : ''}`;

      const response = await fetch(url);
      if (!response.ok) {
        throw new Error(`HTTP ${response.status}: ${response.statusText}`);
      }

      return await response.json();
    }

    // 获取跟踪的赛事
    async getTrackedEvents() {
      const response = await fetch(`${this.config.apiUrl}/events`);
      if (!response.ok) {
        throw new Error(`HTTP ${response.status}: ${response.statusText}`);
      }

      return await response.json();
    }

    // 获取特定赛事的消息
    async getEventMessages(eventId) {
      const response = await fetch(`${this.config.apiUrl}/events/${eventId}/messages`);
      if (!response.ok) {
        throw new Error(`HTTP ${response.status}: ${response.statusText}`);
      }

      return await response.json();
    }

    // 获取统计信息
    async getStats() {
      const response = await fetch(`${this.config.apiUrl}/stats`);
      if (!response.ok) {
        throw new Error(`HTTP ${response.status}: ${response.statusText}`);
      }

      return await response.json();
    }
  }

  // 导出到全局
  window.UOFClient = UOFClient;

  window.createUOFClient = function(config) {
    return new UOFClient(config);
  };

  console.log('UOF Client loaded');

})(window);

