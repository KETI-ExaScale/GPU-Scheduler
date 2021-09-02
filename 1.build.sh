registry="ketidevit"
imagename="gpu-scheduler"
version="v0.1"

#latest golang
#export PATH=$PATH:/usr/local/go/bin && \

#gpu-scheduler이름으로 실행파일 생성
go build -a --ldflags '-extldflags "-static"' -tags netgo -installsuffix netgo . && \

# make image
docker build -t $imagename:$version . && \

# add tag
# docker image에 나의 registry tag를 붙여야 push 가능
docker tag $imagename:$version $registry/$imagename:$version && \

# login
# Docker에 push하기 위해 login
docker login && \

# push image
docker push $registry/$imagename:$version 
