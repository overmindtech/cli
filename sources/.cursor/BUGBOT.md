# Source-Specific Rules

## New Adapters

When reviewing newly created adapters, it is extremely important to ensure that all of the `LinkedItemQueries` that could be added have been added. The way this is done is by looking at the method in which we translate from the cloud provider's data type to an `sdp.Item`. You should look at the definition of the cloud provider's type. This will almost always be a struct with fields, which are quite often other nested structs. What you should do is go through every field in that struct and its children, and see whether it is likely that those fields reference other cloud resources that we could potentially create a link to. Doesn't matter whether or not we have created the adapter for that type of cloud resource yet. We should always create as many links as possible. If it is another cloud resource that we are likely to also create an adapter for at some point.

There are also a couple of generic types that we should always create links for if the attributes are there. These are:

* `ip`: Any attribute that would contain an IP address should create a LinkedItemQueries for an `ip` type. This should always use the scope of global, the method of GET and a query of the IP address itself
* `dns`: any attribute that contains a DNS name should create a LinkedItemQueries for a DNS type.  The type should be `dns`, scope `global`, method SEARCH with the query being the DNS name itself