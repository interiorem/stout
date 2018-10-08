package isolate

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"time"
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
	Cfg MtnCfg
	Db *bolt.DB
}

type Allocation struct {
	Net string
	Hostname string
	Ip string
	Id string
	NetId string
	Box string
	Used bool
}

type PostAllocreq struct {
	Network string `json:"network"`
	Host string `json:"host"`
	Scheduler string `json:"scheduler"`
}

func SaveRename(src, dst string) error {
	ioSrcDb, errSrcOpen := os.Open(src)
	if errSrcOpen != nil {
		return errSrcOpen
	}
	defer ioSrcDb.Close()
	ioDstDb, errDstCreate := os.Create(dst)
	if errDstCreate != nil {
		return errDstCreate
	}
	defer ioDstDb.Close()
	_, errCopy := io.Copy(ioSrcDb, ioDstDb)
	if errCopy != nil {
		return errCopy
	}
	errRemove := os.Remove(src)
	if errRemove != nil {
		return errRemove
	}
	return nil
}

func (c *MtnState) CfgInit(ctx context.Context, cfg *Config) bool {
	c.Cfg.Enable = cfg.Mtn.Enable
	if !c.Cfg.Enable {
		return true
	}
	if cfg.Mtn.Allocbuffer < 1 {
		c.Cfg.Allocbuffer = 4
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

	if len(cfg.Mtn.DbPath) > 1 {
		c.Cfg.DbPath = cfg.Mtn.DbPath
	} else {
		c.Cfg.DbPath = "/run/isolate.mtn.db"
	}
	corruptedBackupPath := "/var/tmp/isolate.mtn.db.corrupted"
	db, err := bolt.Open(c.Cfg.DbPath, 0666, &bolt.Options{Timeout: 10 * time.Second})
	if err != nil {
		log.G(ctx).Errorf("Cant open db inside CfgInit() by calling bolt.Open(), returned: %s", err)
		if s, err := os.Stat(c.Cfg.DbPath); os.IsNotExist(err) {
			log.G(ctx).Errorf("DB file not exist and we cant create new. Err: %s", err)
			return false
		} else if err == nil {
			fSize := s.Size()
			if fSize > 0 {
				if _, err := os.Stat(corruptedBackupPath); err == nil {
					log.G(ctx).Errorf("Corrupted DB backup file exist, nothing to do there.")
					return false
				}
				log.G(ctx).Errorf("DB file exist, size %d and cant be opened. Try to recreate.", fSize)
				errMove := SaveRename(c.Cfg.DbPath, corruptedBackupPath)
				if errMove != nil {
					log.G(ctx).Errorf("Cant move corrupted db file, err: %s", errMove)
					return false
				}
			} else {
				log.G(ctx).Errorf("DB file exist, size %d and cant be opened. Try to delete old.", fSize)
				err := os.Remove(c.Cfg.DbPath)
				if err != nil {
					log.G(ctx).Errorf("Cant delete old db file, err: %s", err)
					return false
				}
			}
			db, err = bolt.Open(c.Cfg.DbPath, 0666, &bolt.Options{Timeout: 10 * time.Second})
			if err != nil {
				log.G(ctx).Errorf("Second try open db is failed, err: %s", err)
				return false
			}
		}
		errDb := db.Update(func(tx *bolt.Tx) error {
			errChan := tx.Check()
			select {
				case errCheck := <-errChan:
					return errCheck
				default:
					return nil
			}
		})
		if errDb != nil {
			log.G(ctx).Errorf("DB fail consistency checks, err: %s", errDb)
			return false
		}
	}
	c.Db = db
	return true
}

func (c *MtnState) PoolInit(ctx context.Context) bool {
	if !c.Cfg.Enable {
		return true
	}
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
		for _, alloc := range allocs {
			b, err := tx.CreateBucketIfNotExists([]byte(netId))
			if err != nil {
				log.G(ctx).Errorf("Cant continue transaction inside PoolInit(), err: %s", err)
				return false
			}
			if b.Get([]byte(alloc.Id)) == nil {
				if buf, err := json.Marshal(Allocation{alloc.Net, alloc.Hostname, alloc.Ip, alloc.Id, netId, "", false}); err != nil {
					log.G(ctx).Errorf("Cant continue transaction inside PoolInit(), err: %s", err)
					return false
				} else if err := b.Put([]byte(alloc.Id), buf); err != nil {
					log.G(ctx).Errorf("Cant continue transaction inside PoolInit(), err: %s", err)
					return false
				}
			}
		}
	}
	if err := tx.Commit(); err != nil {
		log.G(ctx).Errorf("Cant commit transaction inside PoolInit(), err: %s", err)
		return false
	}
	return true
}

func (c *MtnState) GetAllocations(logCtx context.Context) (map[string][]Allocation, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 30 * time.Second)
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
		r[a.Network] = append(r[a.Network], Allocation{a.Porto.Net, a.Porto.Hostname, a.Porto.Ip, a.Id, a.Network, "", false})
	}
	log.G(logCtx).Debugf("GetAllocations() successfull ended with ContentLength size %d.", req.ContentLength)
	return r, nil
}

func (c *MtnState) RequestAllocs(ctx context.Context, netid string) (map[string]Allocation, error) {
	r := make(map[string]Allocation)
	httpCtx, cancel := context.WithTimeout(context.Background(), 30 * time.Second)
	defer cancel()
	jsonBody := PostAllocreq{netid, c.Cfg.Ident, c.Cfg.SchedLabel}
	txtBody, errMrsh := json.Marshal(jsonBody)
	if errMrsh != nil {
		return nil, errMrsh
	}
	log.G(ctx).Debugf("c.Cfg.Allocbuffer inside RequestAllocs() is %d.", c.Cfg.Allocbuffer)
	for i := 0; i < c.Cfg.Allocbuffer; i++ {
		req, errNewReq := http.NewRequest("POST", c.Cfg.Url, bytes.NewReader(txtBody))
		if errNewReq != nil {
			return nil, errNewReq
		}
		for header, value := range c.Cfg.Headers {
			req.Header.Set(header, value)
		}
		req.Header.Set("Content-Type", "application/json")
		req = req.WithContext(httpCtx)
		reqHttp, errDo := http.DefaultClient.Do(req)
		if errDo != nil {
			return nil, errDo
		}
		jsonResp := RawAlloc{}
		decoder := json.NewDecoder(reqHttp.Body)
		errDecode := decoder.Decode(&jsonResp)
		log.G(ctx).Debugf("reqHttp.Body from allocator getted in RequestAllocs(): %s", reqHttp.Body)
		reqHttp.Body.Close()
		if errDecode != nil {
			return nil, errDecode
		}
		log.G(ctx).Debugf("Allocation from allocator getted in RequestAllocs(): %s", jsonResp)
		r[jsonResp.Id] = Allocation{jsonResp.Porto.Net, jsonResp.Porto.Hostname, jsonResp.Porto.Ip, jsonResp.Id, netid, "", false}
	}
	log.G(ctx).Debugf("RequestAllocs() successfull ended with %s.", r)
	return r, nil
}

func (c *MtnState) DbAllocIsFree(ctx context.Context, value []byte) bool {
	var a Allocation
	if err := json.Unmarshal(value, &a); err != nil {
		log.G(ctx).Errorf("DbAllocIsFree() failed on json.Unmarshal()  with error:  %s.", err)
		return false
	}
	if a.Used {
		return false
	}
	return true
}

func (c *MtnState) UsedAllocations(ctx context.Context) ([]Allocation, error) {
	tx, errTx := c.Db.Begin(true)
	var allocs []Allocation
	if errTx != nil {
		return allocs, errTx
	}
	defer tx.Rollback()
	errBkLs := tx.ForEach(func(netId []byte, b *bolt.Bucket) error {
		errAllocsLs := b.ForEach(func(allocId []byte, value []byte) error {
			if c.DbAllocIsFree(ctx, value) {
				return nil
			}
			log.G(ctx).Warnf("Found used alloc for net id %s with id %s: %s", netId, allocId, value)
			var a Allocation
			if errUnmrsh := json.Unmarshal(value, &a); errUnmrsh != nil {
				return errUnmrsh
			}
			allocs = append(allocs, a)
			return nil
		})
		return errAllocsLs
	})
	if errBkLs != nil {
		return allocs, errBkLs
	}
	if errCommit := tx.Commit(); errCommit != nil {
		return allocs, errCommit
	}
	return allocs, nil
}

func (c *MtnState) GetDbAlloc(ctx context.Context, tx *bolt.Tx, netId string, box string) (Allocation, error) {
	b := tx.Bucket([]byte(netId))
	if b == nil {
		return Allocation{}, fmt.Errorf("BUG inside GetDbAlloc()! Backet %s not exist!", netId)
	}
	var a Allocation
	cr := b.Cursor()
	for k, v := cr.First(); k != nil; k, v = cr.Next() {
		if c.DbAllocIsFree(ctx, v) {
			if err := json.Unmarshal(v, &a); err != nil {
				return a, err
			}
			a.Used = true
			a.Box = box
			id := a.Id
			value, errMrsh := json.Marshal(a)
			if errMrsh != nil {
				return a, errMrsh
			}
			errPut := b.Put([]byte(id), value)
			if errPut != nil {
				return a, errPut
			}
			return a, nil
		}
	}
	fcounter, errCnt := c.CountFreeAllocs(ctx, tx, netId)
	log.G(ctx).Errorf("Normaly we must never be in GetDbAlloc() at that point. But ok, lets try fix situation. Free count for that netId %s is %d (possible counter error: %s).", netId, fcounter, errCnt)
	allocs, errAllocs := c.RequestAllocs(ctx, netId)
	if errAllocs != nil {
		log.G(ctx).Errorf("Last hope in GetDbAlloc() failed.")
		return a, errAllocs
	}
	gotcha := false
	b, errCrBk := tx.CreateBucketIfNotExists([]byte(netId))
	if errCrBk != nil {
		return a, errCrBk
	}
	for id, alloc := range allocs {
		if !gotcha {
			alloc.Used = true
			alloc.Box = box
			a = alloc
			gotcha = true
		}
		value, errMrsh := json.Marshal(alloc)
		if errMrsh != nil {
			return a, errMrsh
		}
		errPut := b.Put([]byte(id), value)
		if errPut != nil {
			return a, errPut
		}
	}
	if gotcha {
		return a, nil
	}
	return a, fmt.Errorf("BUG inside GetDbAlloc()... or somewhere! Cant get allocation from DB and cant request more. Clean allocaion: %s.", a)
}

func (c *MtnState) FreeDbAlloc(ctx context.Context, netId string, id string) error {
	tx, errTx := c.Db.Begin(true)
	if errTx != nil {
		log.G(ctx).Errorf("Cant start transaction inside FreeDbAlloc(), err: %s", errTx)
		return errTx
	}
	defer tx.Rollback()
	b := tx.Bucket([]byte(netId))
	if b == nil {
		return fmt.Errorf("BUG inside FreeDbAlloc()! Bucket %s not exist!", netId)
	}
	v := b.Get([]byte(id))
	var a Allocation
	if err := json.Unmarshal(v, &a); err != nil {
		return err
	}
	a.Used = false
	value, errMrsh := json.Marshal(a)
	if errMrsh != nil {
		return errMrsh
	}
	errPut := b.Put([]byte(id), value)
	if errPut != nil {
		return errPut
	}
	if errCommit := tx.Commit(); errCommit != nil {
		return errCommit
	}
	return nil
}

func (c *MtnState) CountFreeAllocs(ctx context.Context, tx *bolt.Tx, netId string) (int, error) {
	b, errBk := tx.CreateBucketIfNotExists([]byte(netId))
	if errBk == nil {
		return 0, errBk
	}
	counter := 0
	e := b.ForEach(func(_, v []byte) error {
		if c.DbAllocIsFree(ctx, v) {
			counter+=1
		}
		return nil
	})
	log.G(ctx).Debugf("CountFreeAllocs() ended for netId %s with count: %d.", netId, counter)
	return counter, e
}

func (c *MtnState) BindAllocs(ctx context.Context, netId string) error {
	if len(netId) == 0 {
		return fmt.Errorf("Len(netId) is zero.")
	}
	log.G(ctx).Debugf("BindAllocs() called with netId %s.", netId)

	tx, errTx := c.Db.Begin(true)
	if errTx != nil {
		log.G(ctx).Errorf("Cant start transaction inside BindAllocs(), err: %s", errTx)
		return errTx
	}
	defer tx.Rollback()
	fCount, errCnt := c.CountFreeAllocs(ctx, tx, netId)
	if errCnt != nil {
		log.G(ctx).Errorf("Cant continue transaction inside BindAllocs(), err: %s", errCnt)
		return errCnt
	}
	if c.Cfg.Allocbuffer > fCount {
		allocs, err := c.RequestAllocs(ctx, netId)
		if err != nil {
			log.G(ctx).Errorf("Cant do c.RequestAllocs(%s) inside BindAllocs(), err: %s", netId, err)
			return err
		}
		log.G(ctx).Debugf("c.RequestAllocs(ctx, %s) end sucessfully with: %s.", netId, allocs)
		b, errBk := tx.CreateBucketIfNotExists([]byte(netId))
		if errBk != nil {
			log.G(ctx).Errorf("Cant create bucket inside BindAllocs(), err: %s", errBk)
			return errBk
		}
		for id, alloc := range allocs {
			value, errMrsh := json.Marshal(alloc)
			if errMrsh != nil {
				log.G(ctx).Errorf("Cant Marshal(%s).", alloc)
				return errMrsh
			}
			errPut := b.Put([]byte(id), value)
			if errPut != nil {
				log.G(ctx).Errorf("Cant b.Put(%s,%s)", id, value)
				return errPut
			}
		}
	}
	if err := tx.Commit(); err != nil {
		return err
	}
	return nil
}

func (c *MtnState) UseAlloc(ctx context.Context, netId string, box string) (Allocation, error) {
	tx, errTx := c.Db.Begin(true)
	if errTx != nil {
		log.G(ctx).Errorf("Cant start transaction inside UseAlloc(), err: %s", errTx)
		return Allocation{}, errTx
	}
	defer tx.Rollback()
	a, e := c.GetDbAlloc(ctx, tx, netId, box)
	log.G(ctx).Debugf("UseAlloc(): a, e: %s, %s.", a, e)
	if e != nil {
		return Allocation{}, e
	}
	if err := tx.Commit(); err != nil {
		return a, err
	}
	return a, nil
}

func (c *MtnState) UnuseAlloc(ctx context.Context, netId string, id string) {
	err := c.FreeDbAlloc(ctx, netId, id)
	if err != nil {
		log.G(ctx).Errorf("BUG inside FreeDbAlloc()! error returned: %s.", err)
	}
	log.G(ctx).Debugf("UnuseAlloc() successfuly for: %s %s.", netId, id)
}

