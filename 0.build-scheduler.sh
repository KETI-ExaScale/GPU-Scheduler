registry="ketidevit2"
image_name="keti-gpu-scheduler"
version="v1.0"

#latest golang
#export PATH=$PATH:/usr/local/go/bin && \
#go mod init scheduler
#go mod vendor
#go mod tidy

#gpu-scheduler binary file
go build -a --ldflags '-extldflags "-static"' -tags netgo -installsuffix netgo . && \

# make image
docker build -t $image_name:$version . && \

# add tag
docker tag $image_name:$version $registry/$image_name:$version && \

# login
docker login && \

# push image
docker push $registry/$image_name:$version 
