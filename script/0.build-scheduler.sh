registry="ketidevit2"
image_name="keti-gpu-scheduler"
version="v2.0"
dir=$( pwd )

#latest golang
#export PATH=$PATH:/usr/local/go/bin && \
#go mod init scheduler
#go mod vendor
#go mod tidy

#gpu-scheduler binary file
# go build -a --ldflags '-extldflags "-static"' -tags netgo -installsuffix netgo . && \
go build -o $dir/../build/_output/bin/$image_name -mod=vendor $dir/../cmd/main.go

# make image
docker build -t $image_name:$version $dir/../build && \

# add tag
docker tag $image_name:$version $registry/$image_name:$version && \

# login
docker login && \

# push image
docker push $registry/$image_name:$version 
