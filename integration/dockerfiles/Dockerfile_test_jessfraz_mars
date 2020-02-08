FROM openjdk:8-alpine

RUN set -x \
	&& apk add --no-cache --virtual .build-deps \
		ca-certificates \
		curl \
	&& curl -sSL "http://courses.missouristate.edu/KenVollmar/mars/MARS_4_5_Aug2014/Mars4_5.jar" -o /mars.jar \
	&& apk del .build-deps

ENTRYPOINT ["java", "-jar", "/mars.jar", "nc"]
