//
// Copyright 2014 Hong Miao. All Rights Reserved.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
// http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package storage

import (
	"sync"
	"time"
	"encoding/json"
	"github.com/garyburd/redigo/redis"
)

type SessionStore struct {
	RS       *RedisStore
	rwMutex  sync.Mutex
}

func NewSessionStore(RS *RedisStore) *SessionStore {
	return &SessionStore {
		RS    : RS,
	}
}

type SessionStoreData struct {
	ClientID      string
	ClientAddr    string
	MsgServerAddr string
	ID            string
	MaxAge        time.Duration
}

func NewSessionStoreData(ClientID string, ClientAddr string, MsgServerAddr string, ID string) *SessionStoreData {
	return &SessionStoreData {
		ClientID      : ClientID,
		ClientAddr    : ClientAddr,
		MsgServerAddr : MsgServerAddr,
		ID            : ID,
	}
}

func (self *SessionStoreData)checkClientID(clientID string) bool {
	return true
}

func (self *SessionStoreData)StoreKey() string {
	return self.ClientID
}

// Get the session from the store.
func (self *SessionStore) Get(k string) (*SessionStoreData, error) {
	self.rwMutex.Lock()
	defer self.rwMutex.Unlock()
	key := k
	if self.RS.opts.KeyPrefix != "" {
		key = self.RS.opts.KeyPrefix + ":" + k
	}
	b, err := redis.Bytes(self.RS.conn.Do("GET", key))
	if err != nil {
		return nil, err
	}
	var sess SessionStoreData
	err = json.Unmarshal(b, &sess)
	if err != nil {
		return nil, err
	}
	return &sess, nil
}

// Save the session into the store.
func (self *SessionStore) Set(sess *SessionStoreData) error {
	self.rwMutex.Lock()
	defer self.rwMutex.Unlock()
	b, err := json.Marshal(sess)
	if err != nil {
		return err
	}
	key := sess.ClientID
	if self.RS.opts.KeyPrefix != "" {
		key = self.RS.opts.KeyPrefix + ":" + sess.ClientID
	}
	ttl := sess.MaxAge
	if ttl == 0 {
		// Browser session, set to specified TTL
		ttl = self.RS.opts.BrowserSessServerTTL
		if ttl == 0 {
			ttl = 2 * 24 * time.Hour // Default to 2 days
		}
	}
	_, err = self.RS.conn.Do("SETEX", key, int(ttl.Seconds()), b)
	if err != nil {
		return err
	}
	return nil
}

// Delete the session from the store.
func (self *SessionStore) Delete(id string) error {
	self.rwMutex.Lock()
	defer self.rwMutex.Unlock()
	key := id
	if self.RS.opts.KeyPrefix != "" {
		key = self.RS.opts.KeyPrefix + ":" + id
	}
	_, err := self.RS.conn.Do("DEL", key)
	if err != nil {
		return err
	}
	return nil
}
// Clear all sessions from the store. Requires the use of a key
// prefix in the store options, otherwise the method refuses to delete all keys.
func (self *SessionStore) Clear() error {
	self.rwMutex.Lock()
	defer self.rwMutex.Unlock()
	vals, err := self.getSessionKeys()
	if err != nil {
		return err
	}
	if len(vals) > 0 {
		self.RS.conn.Send("MULTI")
		for _, v := range vals {
			self.RS.conn.Send("DEL", v)
		}
		_, err = self.RS.conn.Do("EXEC")
		if err != nil {
			return err
		}
	}
	return nil
}
// Get the number of session keys in the store. Requires the use of a
// key prefix in the store options, otherwise returns -1 (cannot tell
// session keys from other keys).
func (self *SessionStore) Len() int {
	self.rwMutex.Lock()
	defer self.rwMutex.Unlock()
	vals, err := self.getSessionKeys()
	if err != nil {
		return -1
	}
	return len(vals)
}
func (self *SessionStore) getSessionKeys() ([]interface{}, error) {
	self.rwMutex.Lock()
	defer self.rwMutex.Unlock()
	if self.RS.opts.KeyPrefix != "" {
		return redis.Values(self.RS.conn.Do("KEYS", self.RS.opts.KeyPrefix+":*"))
	}
	return nil, ErrNoKeyPrefix
}