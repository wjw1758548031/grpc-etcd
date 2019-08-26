package main

import (
	"fmt"
	"log"
	"time"

	"github.com/garyburd/redigo/redis"
)

type Lock struct {
	resource string
	token    string
	conn     redis.Conn
	timeout  int
}

func (lock *Lock) tryLock() (ok bool, err error) {
	_, err = redis.String(lock.conn.Do("SET", lock.key(), lock.token, "EX", int(lock.timeout), "NX"))
	if err == redis.ErrNil {
		// The lock was not successful, it already exists.
		return false, nil
	}
	if err != nil {
		return false, err
	}
	return true, nil
}

func (lock *Lock) Unlock() (err error) {
	_, err = lock.conn.Do("del", lock.key())
	return
}

func (lock *Lock) key() string {
	return fmt.Sprintf("redislock:%s", lock.resource)
}

func (lock *Lock) AddTimeout(ex_time int64) (ok bool, err error) {
	//查看还剩多少秒
	ttl_time, err := redis.Int64(lock.conn.Do("TTL", lock.key()))
	if err != nil {
		log.Fatal("redis get failed:", err)
	}
	if ttl_time > 0 {
		_, err := redis.String(lock.conn.Do("SET", lock.key(), lock.token, "EX", int(ttl_time+ex_time)))
		if err == redis.ErrNil {
			return false, nil
		}
		if err != nil {
			return false, err
		}
	}
	return false, nil
}

func TryLock(conn redis.Conn, resource string, token string, DefaulTimeout int) (lock *Lock, ok bool, err error) {
	return TryLockWithTimeout(conn, resource, token, DefaulTimeout)
}

func TryLockWithTimeout(conn redis.Conn, resource string, token string, timeout int) (lock *Lock, ok bool, err error) {
	lock = &Lock{resource, token, conn, timeout}

	ok, err = lock.tryLock()

	if !ok || err != nil {
		lock = nil
	}

	return
}

func main() {
	fmt.Println("start")
	DefaultTimeout := 10

	pool := redis.Pool{
		MaxIdle: 3,
		IdleTimeout: 240 * time.Second,
		Dial: func () (redis.Conn, error) {
			c, err := redis.Dial("tcp", "10.10.0.145:6379")
			if err != nil {
				return nil, err
			}
			if _, err := c.Do("AUTH", "YkBEBxr5BpyFd9wB"); err != nil {
				c.Close()
				return nil, err
			}
			//if _, err := c.Do("SELECT",1); err != nil {
			// c.Close()
			// return nil, err
			//}
			return c, err
		},
		TestOnBorrow: func(c redis.Conn, t time.Time) error {
			_, err := c.Do("PING")
			return err
		},
	}
	conn,err := pool.Dial()
	if err != nil {
		log.Fatal("Error while attempting lock err:",err)
		return
	}

	lock, ok, err := TryLock(conn, "xiaoru.cc", "token", int(DefaultTimeout))
	if err != nil {
		log.Fatal("Error while attempting lock")
		return
	}
	if !ok {
		log.Fatal("Lock")
		return
	}
	//防止程序终止，每次有访问时都会添加10秒
	lock.AddTimeout(10)

	time.Sleep(time.Duration(DefaultTimeout) * time.Second)
	fmt.Println("b11111")
	fmt.Println("b22222")
	time.Sleep(time.Duration(DefaultTimeout) * time.Second)
	fmt.Println("end")
	defer lock.Unlock()
}