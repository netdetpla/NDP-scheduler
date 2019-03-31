package main

import (
	"github.com/docker/docker/api/types"
	"github.com/docker/docker/client"
	"golang.org/x/net/context"
	"os"
	"strconv"
)

var loadOpt = make(chan imageInfo, 10)

func imageLoaderWithShell(imageConf *image) (err error){
	ctx := context.Background()
	cli, err := client.NewClientWithOpts(client.FromEnv)
	if err != nil {
		return
	}
	for {
		i := <- loadOpt
		// 读文件
		imageFile, err := os.Open(imageConf.Path + i.fileName)
		if err != nil {
			// TODO 错误处理
			continue
		}
		// docker load
		_, err = cli.ImageLoad(ctx, imageFile, false)
		if err != nil {
			// TODO 错误处理
			continue
		}
		// docker tag
		// TODO 测试参数是否正确
		portStr := strconv.FormatFloat(imageConf.RepoPort, 'f', -1, 64)
		err = cli.ImageTag(ctx,
			i.imageName + ":" + i.tag,
			"localhost:" + portStr + "/" + i.imageName + ":" + i.tag)
		// docker push
		_, err = cli.ImagePush(ctx, "localhost:" + portStr + "/" + i.imageName + ":" + i.tag, types.ImagePushOptions{})
		if err != nil {
			// TODO 错误处理
			continue
		}
		scanOpt <- dbOpt{"loaded", []string{i.imageName, i.tag}}
	}
}
