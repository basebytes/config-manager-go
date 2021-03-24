package config

import (
	"log"
	"os"
	"path/filepath"
	"strings"
	"sync"

	"github.com/fsnotify/fsnotify"
	"github.com/mitchellh/mapstructure"
)

const (
	WriteOrCreateMask =fsnotify.Write|fsnotify.Create
	RenameOrRemoveMask=fsnotify.Rename|fsnotify.Remove
)

type CfgManager struct{
	defaults map[string]interface{}
	loader Loader
	filePath string
	keyDelim string
	autoCreate bool
	recoverIfRemove bool
	watchConfigFile bool
	onConfigChange func(fsnotify.Event)
}

type Option func(m *CfgManager)

type Loader interface {
	Load(path string,v interface{}) error
	Save(path string,v interface{}) error
}

func New(loader Loader,filePath string,options ...Option) *CfgManager{
	manager:=new(CfgManager)
	manager.loader=loader
	manager.autoCreate=true
	manager.keyDelim="."
	manager.recoverIfRemove=true
	manager.watchConfigFile=true
	manager.filePath=filePath
	manager.defaults=make(map[string]interface{})
	for _,option:=range options{
		option(manager)
	}
	return manager
}

// file change event
func OnConfigChange(f func(fsnotify.Event)) Option{
	return func(m *CfgManager) {
		m.onConfigChange=f
	}
}
// watch config changes,default:true
func WatchConfigFile(v bool)Option{
	return func(m *CfgManager) {
		m.watchConfigFile=v
	}
}

//must set before WithDefaultConfig and WithDefaultConfigs
func KeyDelim(delim string) Option{
	return func(m *CfgManager) {
		m.keyDelim=delim
	}
}

//default:true
func CreateIfNotExists(v bool) Option{
	return func(m *CfgManager) {
		m.autoCreate=v
	}
}

// override configuration in v
func WithDefaultConfig(k string,v interface{}) Option{
	return func(m *CfgManager) {
		m.setDefault(k,v)
	}
}

// override configuration in v
func WithDefaultConfigs(configs map[string]interface{}) Option{
	return func(m *CfgManager) {
		for k,v:=range configs{
			m.setDefault(k,v)
		}
	}
}

func (m *CfgManager) ReadConfig(v interface{}) error {
	if err:=mapstructure.Decode(m.defaults,&v);err!=nil{
		return err
	}
	_,err:=os.Stat(m.filePath)
	if os.IsNotExist(err)&&m.autoCreate{
		if err=m.createFile();err==nil{
			err=m.loader.Save(m.filePath,&v)
		}
	}
	if err!=nil{
		return err
	}
	if err=m.loader.Load(m.filePath,v);err!=nil{
		return err
	}
	if m.watchConfigFile{
		m.watchConfig(&v)
	}

	return nil
}

func (m *CfgManager)setDefault(key string,value interface{}){
	path:=strings.Split(key,m.keyDelim)
	valueMap:=search(m.defaults,path[0:len(path)-1])
	valueMap[path[len(path)-1]]=value
}

func (m *CfgManager) createFile() error{
	dir:=filepath.Dir(m.filePath)
	if _,err:=os.Stat(dir);os.IsNotExist(err){
		if err=os.MkdirAll(dir,0775);err!=nil{
			return err
		}
	}
	return nil
}

func (m *CfgManager) watchConfig(v interface{}){
	initWG :=sync.WaitGroup{}
	initWG.Add(1)
	go func() {
		watcher,err:=fsnotify.NewWatcher()
		if err !=nil{
			log.Fatal(err)
		}
		defer watcher.Close()
		configFile:=filepath.Clean(m.filePath)
		configDir,_:=filepath.Split(configFile)
		realConfigFile,_:=filepath.EvalSymlinks(m.filePath)

		eventWG:=sync.WaitGroup{}
		eventWG.Add(1)
		go func() {
			for{
				select {
					case event,OK := <-watcher.Events:
						if !OK{
							eventWG.Done()
							return
						}
						currentConfigFile,_:=filepath.EvalSymlinks(m.filePath)
						if eventName:=filepath.Clean(event.Name);(eventName==configFile&&event.Op==fsnotify.Write)||
							(currentConfigFile!=""&&currentConfigFile!=realConfigFile){
							realConfigFile=currentConfigFile
							if err=m.loader.Load(m.filePath,&v);err!=nil{
								log.Printf("reload config file :%s error:%s\n",m.filePath,err)
							}
							if m.onConfigChange!=nil{
								m.onConfigChange(event)
							}
						}else if eventName==configFile &&
							event.Op&RenameOrRemoveMask!=0{
							if err=m.loader.Save(m.filePath,&v);err!=nil{
								log.Printf("recover config file :%s error:%s\n",m.filePath,err)
							}
							if m.onConfigChange!=nil{
								m.onConfigChange(event)
							}
						}
					case err,OK := <-watcher.Errors:
						if OK{
							log.Printf("watcher error:%s\n",err)
						}
						eventWG.Done()
						return
				}
			}
		}()
		if err=watcher.Add(configDir);err!=nil{
			log.Fatal(err)
		}
		initWG.Done()
		eventWG.Wait()
	}()

	initWG.Wait()
}

func search(m map[string]interface{},path []string) map[string]interface{}{
	tmp:=m
	for _,p:=range path{
		subMap,OK:=tmp[p]
		if !OK{
			newMap:=make(map[string]interface{})
			tmp[p]=newMap
			tmp=newMap
			continue
		}
		subMap2,OK:=subMap.(map[string]interface{})
		if !OK{
			subMap2=make(map[string]interface{})
			tmp[p]=subMap2
		}
		tmp=subMap2
	}
	return tmp
}



