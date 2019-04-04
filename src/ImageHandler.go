package main

import (
	"fmt"
	"github.com/docker/docker/api/types"
	"github.com/docker/docker/client"
	"golang.org/x/net/context"
	"os"
)

var loadOpt = make(chan imageInfo, 10)

func imageLoader(imageConf *image) {
	fmt.Println("ih start")
	ctx := context.Background()
	cli, err := client.NewClientWithOpts(client.WithVersion("1.39"))
	if err != nil {
		return
	}
	for {
		i := <- loadOpt
		// 读文件
		imageFile, err := os.Open(imageConf.Path + i.fileName)
		if err != nil {
			// TODO 错误处理
			fmt.Print(err.Error())
			continue
		}
		// docker load
		_, err = cli.ImageLoad(ctx, imageFile, false)
		if err != nil {
			// TODO 错误处理
			fmt.Print(err.Error())
			continue
		}
		// docker tag
		// TODO 测试参数是否正确
		err = cli.ImageTag(ctx,
			i.imageName + ":" + i.tag,
			"localhost:" + imageConf.RepoPort + "/" + i.imageName + ":" + i.tag)
		// docker push
		_, err = cli.ImagePush(ctx, "localhost:" + imageConf.RepoPort + "/" + i.imageName + ":" + i.tag, types.ImagePushOptions{All: true, RegistryAuth: "123"})
		if err != nil {
			fmt.Print(err.Error())
			// TODO 错误处理
			continue
		}
		scanOpt <- dbOpt{"loaded", []string{i.imageName, i.tag}}
	}
}
