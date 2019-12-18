package main

import (
	"fmt"
	"github.com/docker/docker/api/types"
	"github.com/docker/docker/client"
	"golang.org/x/net/context"
	"os"
)

var loadOpt = make(chan imageInfo, 10)

func imageInfo2String(i imageInfo) (s string) {
	s = fmt.Sprintf("image name: %s, tag: %s, file name: %s", i.imageName, i.tag, i.fileName)
	return
}

func imageLoader(imageConf *image) {
	log.Notice("image loader started.")
	ctx := context.Background()
	cli, err := client.NewClientWithOpts(client.WithVersion("1.39"))
	if err != nil {
		log.Error(err.Error())
		return
	}
	for {
		i := <-loadOpt
		log.Debugf(imageInfo2String(i))
		// 读文件
		imageFile, err := os.Open(imageConf.Path + i.fileName)
		if err != nil {
			log.Warning(err.Error())
			continue
		}
		// docker load
		_, err = cli.ImageLoad(ctx, imageFile, false)
		if err != nil {
			log.Warning(err.Error())
			continue
		}
		// docker tag
		err = cli.ImageTag(ctx,
			i.imageName+":"+i.tag,
			"localhost:"+imageConf.RepoPort+"/"+i.imageName+":"+i.tag)
		if err != nil {
			log.Warning(err.Error())
			continue
		}
		// docker push
		_, err = cli.ImagePush(
			ctx,
			"localhost:"+imageConf.RepoPort+"/"+i.imageName+":"+i.tag,
			types.ImagePushOptions{All: true, RegistryAuth: "123"})
		if err != nil {
			log.Warning(err.Error())
			continue
		}
		log.Info("push success")
		scanOpt <- dbOpt{"loaded", []string{i.imageName, i.tag}}
		err = os.Remove(imageConf.Path + i.fileName)
		if err != nil {
			log.Warning(err.Error())
			continue
		}
	}
}
