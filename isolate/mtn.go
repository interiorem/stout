package isolate

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"time"
	"sync"
        bolt "go.etcd.io/bbolt"
	"github.com/noxiouz/stout/pkg/log"
)

type RawAlloc struct {
	Id string `json:"id"`
	Porto RawPorto `json:"porto"`
	Network string `json:"network"`
}

type RawPorto struct {
	Hostname string `json:"hostname"`
	Ip string `json:"ip"`
	Net string `json:"net"`
	Id string `json:",omitempty"`
}

type AllocError struct {
        Type string `json:"type,omitempty"`
        Message []string `json:"message"`
        Cause ErrorCause `json:"cause"`
}

type ErrorCause struct {
        Type string `json:"type,omitempty"`
        Message []string `json:"message"`
        Cause string `json:"cause,omitempty"`
}

type RawAllocs []RawAlloc

type AllocAnswer string

type MtnCfg struct {
	Enable bool
	Allocbuffer int
	Url string
	Ident string
	SchedLabel string
	Headers map[string]string
	DbPath string
}

type MtnState struct {
	sync.Mutex
	Cfg MtnCfg
	Pool MtnPool
	Db *bolt.DB
}

type MtnPool map[string]*IdState

type IdState struct {
	sync.Mutex
	Reserved int
	Allocations map[string]Allocation
}

type Allocation struct {
	Net string
	Hostname string
	Ip string
	Id string
	Used bool
}

type PostAllocreq struct {
	Network string `json:"network"`
	Host string `json:"host"`
	Scheduler string `json:"scheduler"`
}

func (c *MtnState) CfgInit(ctx context.Context, cfg *Config) bool {
	c.Cfg.Enable = cfg.Mtn.Enable
	if !c.Cfg.Enable {
		return true
	}
	if cfg.Mtn.Allocbuffer < 1 {
		c.Cfg.Allocbuffer = 3
	} else {
		c.Cfg.Allocbuffer = cfg.Mtn.Allocbuffer
	}
	if len(cfg.Mtn.Label) == 0 {
		fqdn, err := os.Hostname()
		if err != nil {
			log.G(ctx).Errorf("Cant get hostname inside CfgInit() by calling os.Hostname(), returned: %s", err)
			return false
		}
		c.Cfg.SchedLabel = fqdn
	} else {
		c.Cfg.SchedLabel = cfg.Mtn.Label
	}
	if len(cfg.Mtn.Ident) == 0 {
		c.Cfg.Ident = c.Cfg.SchedLabel
	} else {
		c.Cfg.Ident = cfg.Mtn.Ident
	}
	c.Cfg.Url = cfg.Mtn.Url
	c.Cfg.Headers = cfg.Mtn.Headers

	db, err := bolt.Open(cfg.Mtn.DbPath, 0666, &bolt.Options{Timeout: 10 * time.Second})
	if err != nil {
		log.G(ctx).Errorf("Cant open db inside CfgInit() by calling bolt.Open(), returned: %s", err)
		return false
	}
	c.Db = db

	return true
}

func (c *MtnState) PoolInit(ctx context.Context) bool {
	if !c.Cfg.Enable {
		return true
	}
	c.Pool = make(map[string]*IdState)
	allAllocs, err := c.GetAllocations(ctx)
	if err != nil {
		log.G(ctx).Errorf("Cant init pool inside PoolInit(), err: %s", err)
		return false
	}

	tx, err := c.Db.Begin(true)
	if err != nil {
		log.G(ctx).Errorf("Cant start transaction inside PoolInit(), err: %s", err)
		return false
	}
	defer tx.Rollback()

	for netId, allocs := range allAllocs {
		cState :=  IdState{
			Reserved: 0,
			Allocations: make(map[string]Allocation),
		}

		for _, alloc := range allocs {
			b, err := tx.CreateBucketIfNotExists([]byte(netId))
			if err != nil {
				log.G(ctx).Errorf("Cant continue transaction inside PoolInit(), err: %s", err)
				return false
			}
			if b.Get([]byte(alloc.Id)) == nil {
				if buf, err := json.Marshal(Allocation{alloc.Net, alloc.Hostname, alloc.Ip, alloc.Id, false}); err != nil {
					log.G(ctx).Errorf("Cant continue transaction inside PoolInit(), err: %s", err)
					return false
				} else if err := b.Put([]byte(alloc.Id), buf); err != nil {
					log.G(ctx).Errorf("Cant continue transaction inside PoolInit(), err: %s", err)
					return false
				}
			}

			cState.Allocations[alloc.Id] = Allocation{alloc.Net, alloc.Hostname, alloc.Ip, alloc.Id, false}
		}
		c.Pool[netId] = &cState
	}
	if err := tx.Commit(); err != nil {
		log.G(ctx).Errorf("Cant commit transaction inside PoolInit(), err: %s", err)
		return false
	}
	return true
}

func (c *MtnState) GetAllocations(logCtx context.Context) (map[string][]Allocation, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 20 * time.Second)
	defer cancel()
	req, nrErr := http.NewRequest("GET", c.Cfg.Url + "?scheduler=" + c.Cfg.SchedLabel, nil)
	if nrErr != nil {
		return nil, nrErr
	}
	for header, value := range c.Cfg.Headers {
		req.Header.Set(header, value)
	}
	req = req.WithContext(ctx)
	rh, doErr := http.DefaultClient.Do(req)
	if doErr != nil {
		if rh.StatusCode == 400 {
			errResp := AllocError{}
			decoder := json.NewDecoder(rh.Body)
			rErr := decoder.Decode(&errResp)
			if rErr != nil {
				return nil, fmt.Errorf(
					"Cant allocate. Request error: %s. Internal error: %s. Caused: %s.",
					doErr,
					errResp.Message[0],
					errResp.Cause.Message[0],
				)
			}
			return nil, fmt.Errorf(
				"Cant allocate. Request error: %s. Body parse error: %s.",
				doErr,
				rErr,
			)
		}
		return nil, doErr
	}
	defer rh.Body.Close()
	r := make(map[string][]Allocation)
	jresp := []RawAlloc{}
	decoder := json.NewDecoder(rh.Body)
	rErr := decoder.Decode(&jresp)
	if rErr != nil {
		return nil, rErr
	}
	for _, a := range jresp {
		r[a.Network] = append(r[a.Network], Allocation{a.Porto.Net, a.Porto.Hostname, a.Porto.Ip, a.Id, false})
	}
	log.G(logCtx).Debugf("GetAllocations() successfull ended with ContentLength size %d.", req.ContentLength)
	return r, nil
}

func (c *MtnState) RequestAllocs(ctx context.Context, netid string) (map[string]Allocation, error) {
	r := make(map[string]Allocation)
	ctx, cancel := context.WithTimeout(context.Background(), 20 * time.Second)
	defer cancel()
	jsonBody := PostAllocreq{netid, c.Cfg.Ident, c.Cfg.SchedLabel}
	txtBody, mrshErr := json.Marshal(jsonBody)
	if mrshErr != nil {
		return nil, mrshErr
	}
	for i := 0; i < c.Cfg.Allocbuffer; i++ {
		req, nrErr := http.NewRequest("POST", c.Cfg.Url, bytes.NewReader(txtBody))
		if nrErr != nil {
			return nil, nrErr
		}
		for header, value := range c.Cfg.Headers {
			req.Header.Set(header, value)
		}
		req.Header.Set("Content-Type", "application/json")
		req = req.WithContext(ctx)
		rh, doErr := http.DefaultClient.Do(req)
		if doErr != nil {
			return nil, doErr
		}
		jresp := RawAlloc{}
	        decoder := json.NewDecoder(rh.Body)
		rErr := decoder.Decode(&jresp)
		rh.Body.Close()
		if rErr != nil {
			return nil, rErr
		}
		r[jresp.Id] = Allocation{jresp.Porto.Net, jresp.Porto.Hostname, jresp.Porto.Ip, jresp.Id, false}
	}
	log.G(ctx).Debugf("RequestAllocs() successfull ended with %s.", r)
	return r, nil
}

func (c *MtnState) BindAllocs(ctx context.Context, netId string) error {
	if len(netId) == 0 {
		return fmt.Errorf("Len(netId) is zero.")
	}
	c.Lock()
	log.G(ctx).Debugf("BindAllocs() called with netId %s.", netId)
	defer c.Unlock()
	if _, p := c.Pool[netId]; p {
		c.Pool[netId].Lock()
		defer c.Pool[netId].Unlock()
		if (len(c.Pool[netId].Allocations) - c.Pool[netId].Reserved) > c.Cfg.Allocbuffer {
			c.Pool[netId].Reserved += c.Cfg.Allocbuffer
			log.G(ctx).Debugf("BindAllocs() ended with c.Pool[netId]: %s.", c.Pool[netId])
			return nil
		} else {
			allocs, reqErr := c.RequestAllocs(ctx, netId)
			if reqErr != nil {
				return reqErr
			}
			c.Pool[netId].Reserved += c.Cfg.Allocbuffer
			for id, alloc := range allocs {
				c.Pool[netId].Allocations[id] = alloc
			}
			log.G(ctx).Debugf("BindAllocs() ended with c.Pool[netId]: %s.", c.Pool[netId])
			return nil
		}
	} else {
		allocs, reqErr := c.RequestAllocs(ctx, netId)
		if reqErr != nil {
			return reqErr
		}
		newPool := IdState{*new(sync.Mutex), c.Cfg.Allocbuffer, allocs}
		c.Pool[netId] = &newPool
		log.G(ctx).Debugf("BindAllocs() ended with c.Pool[netId]: %s.", c.Pool[netId])
		return nil
	}
}

func (c *MtnState) UseAlloc(ctx context.Context, netId string) (Allocation, error) {
	c.Lock()
	newPool := c.Pool[netId]
	log.G(ctx).Debugf("UseAlloc() called with netId: %s; and c.Pool[netId] is: %s", netId, c.Pool[netId])
	newAllocs := make(map[string]Allocation)
	found := false
	var rId string
	for id, alloc := range newPool.Allocations {
		if found {
			newAllocs[id] = alloc
			continue
		}
		if !newPool.Allocations[id].Used {
			alloc.Used = true
			newAllocs[id] = alloc
			rId = id
			found = true
		}
	}
	if found {
		newPool.Allocations = newAllocs
		c.Pool[netId] = newPool
		log.G(ctx).Debugf("UseAlloc() successfull ended for netId: %s with c.Pool[netId]: %s", netId, c.Pool[netId])
		c.Unlock()
		return c.Pool[netId].Allocations[rId], nil
	}
	c.Unlock()
	return Allocation{}, fmt.Errorf("BUG! Cant find free alloc in %s netid! newAllocs map is: %s", netId, newAllocs)
}

func (c *MtnState) UnuseAlloc(ctx context.Context, netId string, id string) {
	c.Lock()
	newPool := c.Pool[netId]
	log.G(ctx).Debugf("UnuseAlloc() called with netId %s and id %s and c.Pool[netId]: %s.", netId, id, c.Pool[netId])
	newAllocs := make(map[string]Allocation)
	for cId, alloc := range newPool.Allocations {
		if cId != id {
			newAllocs[cId] = alloc
			continue
		}
		alloc.Used = false
		newAllocs[id] = alloc
	}
	newPool.Allocations = newAllocs
	c.Pool[netId] = newPool
	log.G(ctx).Debugf("UnuseAlloc() ended with netId %s and id %s and c.Pool[netId]: %s.", netId, id, c.Pool[netId])
	c.Unlock()
}


