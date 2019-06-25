package cookiemanage

import (
	"net/http"
	"sync/atomic"
)

const (
	statusRun   int32 = iota //运行中的状态
	statusClose              //关闭的状态
)

// CookieManage cookie管理器
type CookieManage struct {
	cookie []*http.Cookie //所有请求都带上这些cookie

	status int32 //对象的状态

	cookieChan chan datachan //修改或增加的cookie的通道

	close chan chan bool //关闭操作cookie列表的线程
}

type datachan struct {
	carr    []*http.Cookie
	endchan chan bool
}

// NewCookieManage 创建新的cookie管理器
func NewCookieManage() *CookieManage {
	ckm := CookieManage{
		cookie:     nil,
		cookieChan: make(chan datachan, 1000),
		close:      make(chan chan bool),
	}

	go ckm.start()

	return &ckm
}

// Close 关闭cookie管理器
func (ckm *CookieManage) Close() {
	status := atomic.SwapInt32(&ckm.status, statusClose)
	//表示已经关闭
	if status == statusClose {
		return
	}

	closechan := make(chan bool)
	//关闭处理接受数据的线程
	ckm.close <- closechan
	<-closechan
}

//Store 修改获取增加cookie
func (ckm *CookieManage) Store(ckarr []*http.Cookie) {
	//判断对象是否关闭
	status := atomic.LoadInt32(&ckm.status)
	//表示已经关闭
	if status == statusClose {
		return
	}

	endchan := make(chan bool)
	dc := datachan{
		carr:    ckarr,
		endchan: endchan,
	}
	ckm.cookieChan <- dc
	<-endchan
}

// Range 遍历cookie
func (ckm *CookieManage) Range(callback func(cookie *http.Cookie) bool) {
	for _, v := range ckm.cookie {
		if !callback(v) {
			break
		}
	}
}

//增加或者修改cooki列表的线程
func (ckm *CookieManage) start() {
	ckm.status = statusRun

	for {
		select {
		case closechan := <-ckm.close:
			defer func() {
				closechan <- true
			}()
			return
		case ckarr := <-ckm.cookieChan:
			ckm.addOrUpdate(ckarr.carr)
			ckarr.endchan <- true
		}
	}
}

//修改cookie列表
func (ckm *CookieManage) addOrUpdate(ckarr []*http.Cookie) {
	//遍历需要更新或者增加cookie
	for _, cookie := range ckarr {
		add := true
		for k, v := range ckm.cookie {
			if v.Name == cookie.Name {
				ckm.cookie[k] = cookie
				add = false
				break
			}
		}

		if add {
			ckm.cookie = append(ckm.cookie, cookie)
		}
	}
}
