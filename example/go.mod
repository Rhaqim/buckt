module example

go 1.24.0

require (
	github.com/Rhaqim/buckt v1.0.2
	github.com/Rhaqim/buckt/client/web v1.0.2
	github.com/Rhaqim/buckt/cloud/aws v1.0.2
	github.com/Rhaqim/buckt/cloud/azure v1.0.2
	github.com/Rhaqim/buckt/cloud/gcp v1.0.2
)

// add module from src folder
replace github.com/Rhaqim/buckt => ../

replace github.com/Rhaqim/buckt/client/web => ../client/web

replace github.com/Rhaqim/buckt/cloud/aws => ../cloud/aws

replace github.com/Rhaqim/buckt/cloud/azure => ../cloud/azure

replace github.com/Rhaqim/buckt/cloud/gcp => ../cloud/gcp
