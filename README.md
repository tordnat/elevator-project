# elevator-project

[![Go](https://github.com/tordnat/elevator-project/actions/workflows/build_go.yml/badge.svg)](https://github.com/tordnat/elevator-project/actions/workflows/build_go.yml)

Elevator project for Real-time Programming TTK4145 at NTNU

2.1: Network design questions
-------------------

Before proceeding with any code related to a network module, think about how you would solve these problems, and what you need in order to solve them.

 - Guarantees about elevators:
   - What should happen if one of the nodes loses its network connection?
     A: It should finish its orders, try to reconnect to the network. 
   - What should happen if one of the nodes loses power for a brief moment?
     A: It re-initializes, gets its current order from other elevator nodes and continues operation if all HW checks are OK.
   - What should happen if some unforeseen event causes the elevator to never reach its destination, but communication remains intact?
     A: All orders should have a time-to-completion, after which the order is re-assigned.
   
 - Guarantees about orders:
   - Do all your nodes need to "agree" on a call for it to be accepted? In that case, how is a faulty node handled?
     A: Yes, but faulty nodes are cut off from the network and cannot take any orders. 
   - How can you be sure that a remote node "agrees" on an call?
     A: ACK which includes sent call. 
   - How do you handle losing packets between the nodes?
     A: We send data frequently, track the most recent communtication between the nodes, and have a watchdog for each connection.
   - Do you share the entire state of the current calls, or just the changes as they occur?
     A: Yes, we send all the data since the transfered data is so small. 
     - For either one: What should happen when an elevator re-joins after having been offline?
       A: All queued orders are redistributed.

*Pencil and paper is encouraged! Drawing a diagram/graph of the message pathways between nodes (elevators) will aid in visualizing complexity. Drawing the order of messages through time will let you more easily see what happens when communication fails.*
     
 - Topology:
   - What kind of network topology do you want to implement? Peer to peer? Master slave? Circle? Something else?
     A: Broadcast network
   - In the case of a master-slave configuration: Do you have only one program, or two (a "master" executable and a "slave")?
     - How do you handle a master node disconnecting?
     - Is a slave becoming a master a part of the network module?
   - In the case of a peer-to-peer configuration:
     - Who decides the order assignment?
       A: Everyone, anarchy!
     - What happens if someone presses the same button on two panels at once? Is this even a problem?
       A: It's not an issue to have identical orders different places in the queue, all identical orders will be fulfilled when one of them is fulfilled. 
     
 - Technical implementation and module boundary:
   - Protocols: TCP, UDP, or something else?
      - If you are using TCP: How do you know who connects to who?
        - Do you need an initialization phase to set up all the connections?
      - If you are using UDP broadcast: How do you differentiate between messages from different nodes?
        A: Unique IPs!
      - If you are using a library or language feature to do the heavy lifting - what is it, and does it satisfy your needs?
        A: The GO-Networking module handles UDP Broadcasing. 
   - Do you want to build the necessary reliability into the module, or handle that at a higher level?
     A: We will build the necessary reliability into the module; Go-Networking
   - Is detection (and handling) of things like lost messages or lost nodes a part of the network module?
     A: No.
   - How will you pack and unpack (serialize) data?
     A: We are using the library, soooo Memcpy?
     - Do you use structs, classes, tuples, lists, ...?
     - JSON, XML, plain strings, or just plain memcpy?
     - Is serialization a part of the network module?
