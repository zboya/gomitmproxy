# gomitmproxy

gomitmproxy是想用golang语言实现的[mitmproxy](https://mitmproxy.org/)，主要实现http代理，目前实现了http代理和https抓包功能，差距甚远，加油。


## 可以用来干嘛？

* http代理
* http和https抓包
* 科学上网

## 将来要实现的功能

* 修改http(s)请求
* 重复请求
* 同时监听多端口
* 支持socks5、websocket等协议
* 界面支持终端和网页两种形式

## 安装使用

```bash
    git clone https://github.com/sheepbao/gomitmproxy.git
    cd gomitmproxy 
    ./build.sh 
```

## 例子

* http代理

```bash
gomitmproxy 
```
不带任何参数，表示http代理，默认端口8080,更改端口用 -port 

* http抓包

```bash
gomitmproxy -m 
```

![fetch http](https://raw.githubusercontent.com/sheepbao/gomitmproxy/master/src/doc/goproxy.png)

加 -m 参数，表示抓取http请求和响应

* http代理科学上网

    首先你得有个墙外的服务器，如阿里香港的服务器，为图中的Server，假设其ip地址为：22.222.222.222

```bash
在Server执行:
    gomitmproxy -port 8888
```

```bash
在你自己电脑执行:
    gomitmproxy -port 8080 -raddr 22.222.222.222:8888
```
然后浏览器设置代理，ip为localhost，端口为8080,即可实现科学上网

![proxy](https://raw.githubusercontent.com/sheepbao/gomitmproxy/master/src/doc/proxy.png) 

## 最后

欢迎star和fork，一起学习交流。
## License

The 3-clause BSD License  
- see LICENSE for more details
