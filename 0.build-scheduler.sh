registry="ketidevit2"
imagename="gpu-scheduler"
version="v0.251"

#latest golang
#export PATH=$PATH:/usr/local/go/bin && \
#go mod init scheduler
#go mod vendor
#go mod tidy

#gpu-scheduler binary file
go build -a --ldflags '-extldflags "-static"' -tags netgo -installsuffix netgo . && \

# make image
docker build -t $imagename:$version . && \

# add tag
docker tag $imagename:$version $registry/$imagename:$version && \

# login
docker login && \

# push image
docker push $registry/$imagename:$version 
