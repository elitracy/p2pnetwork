# Metromesh

A p2p-control plane network. Nodes scattered around the city talk to each other and the server to 
manage and distribute data for personal use.

Ideally, I would never have to leave my vpn in the city. I could access my home network from anywhere while
still using public wifi. I'd be able to access all of my projects, data, pictures, movies, etc from 
anywhere in the city independent of whatever network I am on.

## TODO (Foundations)
- [ ] wireguard auth
 - [ ] public key exchange
 - [ ] rotate keys
 - [ ] request signing/verification
- [ ] time pings (for distance graph)
- [ ] basic p2p messages
 - [ ] calculate shortest path between nodes
- [ ] sync messages to server (proof-of-concept)

## TODO (Long term)
- [ ] track bike stats
 - [ ] speed (arduino, phone?)
 - [ ] tilt (arduino/gyroscope, phone?)
 - [ ] mileage (fuely?)
 - [ ] gas (fuely?)
 - [ ] pathing (phone?)
- [ ] sync bike stats with server
 - [ ] upload to nodes (phone?, raspberrypi?)
  - [ ] sync nodes to server
- [ ] map out best paths
 - [ ] configure best routes
 - [ ] bad traffic areas? 
 - [ ] dangerous routes? (potholes, construction, bad interections, etc)
