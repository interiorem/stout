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
}

type MtnState struct {
	sync.Mutex
	Cfg MtnCfg
	Pool MtnPool
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

func (c *MtnState) CfgInit(cfg *Config) bool {
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
			fmt.Println("%s ERROR: Cant get hostname inside CfgInit() by calling os.Hostname(), returned: %s", time.Now().UTC().Format(time.RFC3339), err)
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
	return true
}

func (c *MtnState) PoolInit() bool {
	if !c.Cfg.Enable {
		return true
	}
	c.Pool = make(map[string]*IdState)
	allAllocs, err := c.GetAllocations()
	if err != nil {
		// TODO: maybe need use main logger at this step
		fmt.Println("%s ERROR: Cant init pool inside PoolInit(), err: %s", time.Now().UTC().Format(time.RFC3339), err)
		return false
	}
	for netId, allocs := range allAllocs {
		cState :=  IdState{
			Reserved: 0,
			Allocations: make(map[string]Allocation),
		}
		for _, alloc := range allocs {
			cState.Allocations[alloc.Id] = Allocation{alloc.Net, alloc.Hostname, alloc.Ip, alloc.Id, false}
		}
		c.Pool[netId] = &cState
	}
	return true
}

func (c *MtnState) GetAllocations() (map[string][]Allocation, error) {
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
					"%s ERROR: Cant allocate. Request error: %s. Internal error: %s. Caused: %s.",
					time.Now().UTC().Format(time.RFC3339),
					doErr,
					errResp.Message[0],
					errResp.Cause.Message[0],
				)
			}
			return nil, fmt.Errorf(
				"%s ERROR: Cant allocate. Request error: %s. Body parse error: %s.",
				time.Now().UTC().Format(time.RFC3339),
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
	return r, nil
}

func (c *MtnState) RequestAllocs(netid string) (map[string]Allocation, error) {
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
	return r, nil

}

func (c *MtnState) BindAllocs(netId string) error {
	c.Lock()
	defer c.Unlock()
	if _, p := c.Pool[netId]; p {
		c.Pool[netId].Lock()
		defer c.Pool[netId].Unlock()
		if (len(c.Pool[netId].Allocations) - c.Pool[netId].Reserved) > c.Cfg.Allocbuffer {
			c.Pool[netId].Reserved += c.Cfg.Allocbuffer
			return nil
		} else {
			allocs, reqErr := c.RequestAllocs(netId)
			if reqErr != nil {
				return reqErr
			}
			c.Pool[netId].Reserved += c.Cfg.Allocbuffer
			for id, alloc := range allocs {
				c.Pool[netId].Allocations[id] = alloc
			}
			return nil
		}
	} else {
		allocs, reqErr := c.RequestAllocs(netId)
		if reqErr != nil {
			return reqErr
		}
		newPool := IdState{*new(sync.Mutex), c.Cfg.Allocbuffer, allocs}
		c.Pool[netId] = &newPool
		return nil
	}
}

func (c *MtnState) UseAlloc(netId string) (Allocation, error) {
	c.Lock()
	newPool := c.Pool[netId]
	newallocs := make(map[string]Allocation)
	found := false
	var rId string
	for id, alloc := range newPool.Allocations {
		if found {
			newallocs[id] = alloc
			continue
		}
		if !newPool.Allocations[id].Used {
			alloc.Used = true
			newallocs[id] = alloc
			rId = id
			found = true
		}
	}
	if found {
		newPool.Allocations = newallocs
		c.Pool[netId] = newPool
		c.Unlock()
		return c.Pool[netId].Allocations[rId], nil
	}
	c.Unlock()
	return Allocation{}, fmt.Errorf("BUG! Cant find free alloc in %s netid!", netId)
}

func (c *MtnState) UnuseAlloc(netId string, id string) {
	c.Lock()
	newAllocs := make(map[string]Allocation)
	for cId, alloc := range c.Pool[netId].Allocations {
		if cId == id {
			alloc.Used = false
			newAllocs[id] = alloc
		}
	}
	newPool := IdState{
		Reserved: c.Pool[netId].Reserved,
		Allocations: newAllocs,
	}
	c.Pool[netId] = &newPool
	c.Unlock()
}


