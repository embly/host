

iptables -t nat -I OUTPUT --src 172.17.0.2 --dst 172.217.3.110 -p tcp --dport 8080 -j REDIRECT --to-ports 80



iptables -A INPUT -p tcp -dport 8080 -j ACCEPT
iptables -t nat -A PREROUTING -p tcp –dport 80 -j REDIRECT –to-ports 8080


sudo iptables -t nat -L OUTPUT --line-number
sudo iptables -t nat -D OUTPUT 1

iptables -t nat -A OUTPUT -p tcp -d 172.217.3.110 --dport 8080 -j DNAT --to-destination 172.217.3.110:80




sudo iptables -t nat -A OUTPUT -p tcp -d 192.168.86.30 --dport 80 -j DNAT --to-destination :8080
sudo iptables -t nat -I PREROUTING -p tcp --dport 80 -j REDIRECT --to-ports 8080

sudo iptables -t nat -I PREROUTING -d 192.168.86.30 -p tcp --dport 80 -j REDIRECT --to-ports 8080

https://serverfault.com/questions/704643/steps-for-limiting-outside-connections-to-docker-container-with-iptables


 - the proxy broadcasts on the host, using the host ip addr (it could also be a docker container, but it's good if there's not much else binding on this ip addr, so maybe another local ip?)
 -


sudo iptables -t nat -A OUTPUT -p tcp -d 172.217.3.110 --dport 8080 -j DNAT --to-destination 172.217.3.110:80
sudo iptables -t nat -I PREROUTING -p tcp  --dport 8080 -j REDIRECT --to-ports 80

sudo iptables -t nat -A OUTPUT -s 172.17.0.2/32 -p tcp -d 172.217.3.110 --dport 8080 -j DNAT --to-destination 172.217.3.110:80

sudo iptables -t nat -L PREROUTING --line-number
sudo iptables -t nat -D PREROUTING 1




```bash
LOCAL_IP=192.168.86.30
REQUESTED_PORT=80
DESTINATIO_PORT=8080
CONTAINER_IP=172.17.0.2
sudo iptables -t nat -A OUTPUT -p tcp -d $LOCAL_IP --dport $REQUESTED_PORT -j DNAT --to-destination :$DESTINATIO_PORT
sudo iptables -t nat -I PREROUTING -s $CONTAINER_IP -d $LOCAL_IP -p tcp --dport $REQUESTED_PORT -j REDIRECT --to-ports $DESTINATIO_PORT
```
