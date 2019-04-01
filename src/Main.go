package main

func main()  {
	config := GetConfig()
	go databaseScanner(&config.Database)
	go imageLoader(&config.Image)
	go taskSender(&config.Server)
	select {}
}
