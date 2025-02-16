apt-get update
sudo apt install python2-minimal
apt install build-essential
apt install net-tools

ssh-keygen -t rsa

vim ~/.ssh/config
Host github.com
 Hostname ssh.github.com
 Port 443
 IdentityFile ~/.ssh/git_rsa

sudo apt install mysql-server

vim /etc/mysql/my.cnf
[mysqld]
port=3308

sudo systemctl start mysql.service
sudo systemctl enable mysql.service
mysql_secure_installation

sudo mysql
ALTER USER 'root'@'localhost' IDENTIFIED WITH mysql_native_password BY 'root';

create database ananbay;
CREATE USER 'ananbay'@'%' IDENTIFIED BY 'ananbay';
GRANT ALL PRIVILEGES ON ananbay.* TO 'ananbay'@'%' WITH GRANT OPTION;
GRANT ALL PRIVILEGES ON dev.* TO 'ananbay'@'%' WITH GRANT OPTION;
FLUSH PRIVILEGES;
exit

GRANT ALL PRIVILEGES ON teb.* TO 'ananbay'@'%' WITH GRANT OPTION;
FLUSH PRIVILEGES;
exit

CREATE USER 'redash'@'%' IDENTIFIED WITH mysql_native_password BY 'redash';
GRANT ALL PRIVILEGES ON ananbay.* TO 'redash'@'%' WITH GRANT OPTION;
GRANT ALL PRIVILEGES ON dev.* TO 'redash'@'%' WITH GRANT OPTION;
FLUSH PRIVILEGES;
exit

sudo apt install redis-server

vim /etc/redis/redis.conf
supervised systemd
maxmemory 500000000
requirepass pass
maxmemory-policy allkeys-lru

systemctl start redis

wget https://go.dev/dl/go1.21.10.linux-amd64.tar.gz
tar -xzf go1.21.10.linux-amd64.tar.gz
mv go /usr/local/

vim ~/.bashrc
export GOROOT=/usr/local/go
export PATH=$PATH:$GOROOT/bin

curl https://raw.githubusercontent.com/creationix/nvm/master/install.sh | bash 
source ~/.bashrc   
nvm install 12.13.0
npm install --global yarn

sudo apt install nginx
systemctl start nginx
systemctl enable nginx
 
yarn run build --mode production

sudo add-apt-repository ppa:certbot/certbot

sudo letsencrypt certonly -a webroot -w /var/www/admin/ -d ananbay.com -d www.ananbay.com

vim /etc/systemd/system/ndapi.service

sudo systemctl daemon-reload

sudo systemctl restart ndapi.service