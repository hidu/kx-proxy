


You can deploy it to heroku with the following commands:
```
$ git clone https://git.heroku.com/{your-app-name}.git
$ cd {your-app-name}
$ git remote add kxproxy https://github.com/hidu/kx-proxy.git
$ git pull kxproxy
$ git merge kxproxy/master
$ godep
$ echo "web:"$(basename `pwd`) >Procfile
$ git push origin master
```

proxy client:
https://github.com/hidu/kx-proxy-client

install:
```
go get -u github.com/hidu/kx-proxy-client
```