package cmd

// Send and receive pitcoin protocol messages (commands)
/*
### Commands that we MUST respond to:

ping: When a node receives a ping message, it must respond with a pong message to show it is still connected.
version: When a node receives a version message, it should reply with a verack message.
getaddr: When a node receives a getaddr request, it should reply with an addr message.
getblocks: When a node receives a getblocks request, it should reply with an inv message containing the hashes of the requested blocks.
getheaders: When a node receives a getheaders request, it should respond with a headers message.
getdata: When a node receives a getdata request for specific items, it should respond with the relevant tx, block, or merkleblock messages.


### Commands that we can send to other nodes:

version: Used to notify the other node about your version.
verack: Acknowledgement of the other node's version.
getaddr: Requests address information from the other node.
getblocks: Asks the other node for an inventory of blocks starting from a particular point.
getheaders: Asks the other node for block headers from a certain point.
getdata: Requests specific data items from the other node.
inv: Advertises inventory (blocks or transactions) to the other node.
tx: Broadcasts transactions to other nodes.
mempool: Requests information about unconfirmed transactions.
ping: Used to check if the other node is alive.
filterload, filteradd, filterclear: Used to set and modify Bloom filters, which allow the node to limit the transactions received to only those that concern a subset of all possible transactions, such as those relevant to one or more addresses.
*/

type cmd string

const (
	// commmands that we must respond to
	cmdPing       cmd = "ping"
	cmdVersion    cmd = "version"
	cmdGetAddr    cmd = "getaddr"
	cmdGetBlocks  cmd = "getblocks"
	cmdGetHeaders cmd = "getheaders"
	cmdGetData    cmd = "getdata"
	// commands that we can send to other nodes
	cmdVerack      cmd = "verack"
	cmdInv         cmd = "inv"
	cmdTx          cmd = "tx"
	cmdMempool     cmd = "mempool"
	cmdFilterLoad  cmd = "filterload"
	cmdFilterAdd   cmd = "filteradd"
	cmdFilterClear cmd = "filterclear"
)
