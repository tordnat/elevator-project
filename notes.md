# Notes
Notes about implementation and other useful .

### Order states:
**Unknown**: We have just joined the network, and haven't caught up yet. Any state from the network is accepted.   
**No order**: We haven't recieved any order. Unly valid transition here is to unconfirmed.   
**Unconfirmed**: We have received an order, but it has not been confirmed with the network. The only transition here is to confirmed.   
**Confirmed**: All nodes on the network were/are in the unconfirmed state. We can now light up the button and potentially (depending on HRA) do the order. 
The only transition here is no order. Either because we have done the order, or because the network has done the order (by going to *no order*)   

We can also look at the states as a cyclic counter from 0 to 2, but with unknown as -1.

### Internal representation
We only send our own state to the network. Because of this we need to keep the state of all other elevators internally. (This is not set in stone, but probably gives the smallest code footprint)

### Cab vs hall
We do not differentiate between cab and hall requests, because we want to the same code for all types of requests. 
This may present problems; if cab orders are not properly distributed on the network there may be a large delay for something that we know our own elevator must handle anyways. 
The advantage is that the HRA can then know all cab request and distribute hall requests optimally (assuming it actually does this. Should be cheked). 
We may also need to add extra explicit fault tolerance so that we are certain cab requests are served.

### Non-monotonic counter for messages
For knowing if a message is new or old we attach a non-monotonic counter to every message. It is a uint64, so in theory our elevator system will outlive the sun if we send messages every 10ms.

### Peers 
We can use the peers network module by having the reciever channel in the main for-select and removing/adding nodes in the ElevatorSystem struct depending on peers.

### MVP for distributed system:
Have *n* elevators where instead of executing order, we print out when we have acknowledged the order and decided who shold execute it (basically HRA)

### CAB sync/consensus
We have to have consensus on cabs, same as hall requests because of the spec about not loosing orders.
