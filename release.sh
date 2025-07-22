VERSION=0.20.0
git tag ${VERSION}
git push origin ${VERSION}
export NGC_TARGET="api.ngc.nvidia.com";
export NGC_CLI_API_URL="https://api.ngc.nvidia.com";
export NGC_CLI_API_KEY="ZjRvYm5lbjVrcGhpamZvcTI5NHFnY3Rna3Y6MWJlOWEyYjktZWY1YS00ZjYxLThiMTctMTM3NzI1YTZlZThm"
helm package deploy
ngc registry chart push nv-ngc-devops/nvcf-container-cache:${VERSION} --org nv-ngc-devops

