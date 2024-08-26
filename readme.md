# 分布式中央集权MD5计算

### 简介

分为master和worker节点，master负责接收worker的连接请求以及心跳信息，向worker发送开始/停止工作的命令，节点之间通过gin构建的后端程序以及发送http请求来通信。同时返回前端页面实现网页控制以及所有worker节点的信息查看。worker

### 思路

- worker在未连接状态下持续（每2s）向主节点发送连接请求，目标地址端口在*config.json*文件中保存并使用viper读取
- worker连接成功后持续获取本机的CPU信息并向主机心跳信息用于保持连接状态不丢失
- master持续检测工人的心跳，如果有某个心跳超过10s未更新则关闭与其的连接
- 添加批量启动或停止的按钮，一次性控制所有节点。

### 改为socket通信
- 工人向主机发送消息的type=1，心跳信息
- 工人向主机发送消息的type=2，计算结果及值
- 主机向工人发送消息的type=3，开始或停止工作



### 使用说明
- 启动master节点，后加参数0即可，示例：
   ```bash
   ./run_win.exe 0
   ./run_linux 0  
   ```

- 启动worker节点
  1. 首先编辑`config.json`文件，格式如下：
   host为要远程连接的主机的地址，修改cores值为该工人CPU核心数量，name给本机一个让人易明白的名字
   ```json
   {
      "host_address":"lzh.zzdx.gay", 
      "host_port":55156,
      "cores":8,
      "name":"张三的XX机"
   }
   ```
   2. 直接执行命令，不加任何参数
   ```bash
   ./run_win.exe
   ./run_linux
   ```
   