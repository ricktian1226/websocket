package ucwebsocket

import (
	"fmt"
	"github.com/astaxie/beego"
	"github.com/gorilla/websocket"
	"github.com/pkg/errors"
	. "marco_uc_server/common"
	"sync"
	"time"
)

//websocket管理对象
type Client struct {
	ws *websocket.Conn //websocket连接对象
}

func NewClient(ws *websocket.Conn) *Client {
	return &Client{
		ws: ws,
	}
}

//
var WSClientManager = NewClientManager()

type ClientManager struct {
	clients sync.Map
}

func NewClientManager() *ClientManager {
	return &ClientManager{}
}

////
//func (this *ClientManager) init() {
//
//}

//设置用户的连接信息
func (this *ClientManager) Set(uid uint64, ws *websocket.Conn) {
	this.clients.Store(uid, NewClient(ws))
}

//获取用户的连接信息
func (this *ClientManager) Get(uid uint64) *Client {
	val, ok := this.clients.Load(uid)
	if ok && val != nil {
		return val.(*Client)
	}

	return nil
}

//尝试释放用户之前的websocket
func (this *ClientManager) TryUnset(uid uint64) {
	if client := this.Get(uid); client != nil && client.ws != nil { //如果ws存在，则先关闭
		client.ws.Close()
	}

	this.clients.Delete(uid)
}

func (this *ClientManager) Register(uid uint64, controller beego.Controller) (err error) {
	//将http请求升级到websocket
	ws, err := websocket.Upgrade(controller.Ctx.ResponseWriter, controller.Ctx.Request, nil, Cursvr.WebSocketReadBuffSize, Cursvr.WebSocketWriteBuffSize)
	if _, ok := err.(websocket.HandshakeError); ok {
		return errors.New("非正常 websocket 握手")
	} else if err != nil {
		LOG_FUNC_ERROR("无法建立ws连接 ： %s", err)
		return err
	}

	//将该ws连接加入管理
	this.Set(uid, ws)

	//如果连接关闭，释放对应资源
	defer this.TryUnset(uid)

	//死循环接受消息
	for {
		_, p, errReadMsg := ws.ReadMessage()
		if errReadMsg != nil {
			return
		}

		//publish <- newEvent(models.EVENT_MESSAGE, uname, string(p))
		//todo 将客户端上报的消息解码出来
		//json.Unmarshal(p, )
		LOG_FUNC_DEBUG("客户端 %d 上报的消息 %s", uid, string(p))

		//4test
		this.Broadcast(uid, p)

	}
}

func (this *ClientManager) Broadcast(uid uint64, p []byte) {
	this.clients.Range(func(key, value interface{}) bool {
		//下发消息给客户端
		c := value.(*Client)
		if c != nil {
			c.ws.WriteMessage(websocket.TextMessage, []byte(fmt.Sprintf("收到客户端 %d 的消息 %s", uid, string(p))))
		}
		return true
	})
}

func (this *ClientManager) Print() {
	for {
		time.Sleep(time.Second * 2)
		log := fmt.Sprintln("ws连接列表：")
		this.clients.Range(func(key, value interface{}) bool {
			//下发消息给客户端
			c := value.(*Client)
			if c != nil {
				log += fmt.Sprintf("uid : %d\n", key.(uint64))
			}
			return true
		})

		LOG_FUNC_DEBUG(log)
	}
}

func init() {
	go WSClientManager.Print()
}
