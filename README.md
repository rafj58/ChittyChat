# ChittyChat

To run the project correctly, you must pay attention to the parameters to pass when the client is started. 

The following commands are to be understood if you are located with the terminal inside the main project folder. 
To start the server simply run the command ```go run ./server/server.go``` and the ip address and port to which the server responds will be shown (port can be changed with the -port flag). 

To properly start the client instead we should specify the port on which to start it and the address of the server. The flags to use are -cPort and -sAddr: ```go run ./client/client.go -sAddr <serverAddress> -cPort <clientPort>``` 

If the default server port is changed we will have to add the -sPort flag.

A name can also be provided with the -cName flag to differentiate between clients.
