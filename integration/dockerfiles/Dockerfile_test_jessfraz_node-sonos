FROM node:alpine

RUN apk add --no-cache \
	build-base \
	ca-certificates \
	git \
	python

RUN git clone --depth 1 https://github.com/jishi/node-sonos-http-api.git /opt/app

# install dependencies
WORKDIR /opt/app
RUN npm install --production

EXPOSE 3500/tcp 5005/tcp

CMD ["npm", "start"]
