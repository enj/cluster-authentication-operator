FROM registry.svc.ci.openshift.org/openshift/release:golang-1.12 AS builder
WORKDIR /go/src/github.com/openshift/cluster-authentication-operator
COPY . .
ENV GO_PACKAGE github.com/openshift/cluster-authentication-operator
RUN go build -ldflags "-X $GO_PACKAGE/pkg/version.versionFromGit=$(git describe --long --tags --abbrev=7 --match 'v[0-9]*')" -tags="ocp" -o authentication-operator ./cmd/authentication-operator

FROM registry.svc.ci.openshift.org/openshift/origin-v4.0:base
COPY --from=builder /go/src/github.com/openshift/cluster-authentication-operator/authentication-operator /usr/bin/
COPY manifests /manifests
ENTRYPOINT ["/usr/bin/authentication-operator"]
LABEL io.openshift.release.operator=true
