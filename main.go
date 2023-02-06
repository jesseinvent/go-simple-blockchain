package main

import(
	"fmt"
	"log"
	"net/http"
	"encoding/json"
	"encoding/hex"
	"crypto/md5"
	"crypto/sha256"
	"io"
	"time"

	"github.com/gorilla/mux"
)

type Book struct{
	ID 			string	`json:"id"`
	Title		string	`json:"title"`
	Author 		string	`json:"author"`
	PublishDate	string	`json:"publish_date"`
	ISDN		string	`json:"isbn"`
}

// Blockchain data
type BookCheckout struct{
	BookID			string 	`json:"book_id"`
	User			string	`json:"user"`
	CheckoutDate	string	`json:"checkout_date"`
	IsGenesis		bool	`json:"is_genesis"`
}

type Block struct{
	Position	int
	Data		BookCheckout
	Timestamp	string
	Hash		string
	PrevHash	string
}

type Blockchain struct{
	blocks []*Block;
}
  
var BookBlockchain *Blockchain; 

func (block *Block) generateHash()  {

	// Get json strinf of data
	bytes, _ := json.Marshal(block.Data);
	 
	// Create data to hash with sha256
	data := string(block.Position) + block.Timestamp + string(bytes) + block.PrevHash;

	hash := sha256.New(); 

	hash.Write([]byte(data));

	block.Hash = hex.EncodeToString(hash.Sum(nil));	  
}

func CreateBlock(prevBlock *Block, checkoutItem BookCheckout) *Block {
	block := &Block{};

	block.Position = prevBlock.Position + 1;
	block.Timestamp = time.Now().String();
	block.Data = checkoutItem;
	block.PrevHash = prevBlock.Hash;
	block.generateHash();

	return block;
}

// Add a block to the blockchain
func (blockchain *Blockchain) AddBlock(data BookCheckout) {

	prevBlock := blockchain.blocks[len(blockchain.blocks) - 1];
	
	block := CreateBlock(prevBlock, data);

	if validBlock(block, prevBlock) {
		blockchain.blocks = append(blockchain.blocks, block);
	}

}

func (block *Block) validateHash (hash string) bool {
	block.generateHash();

	if block.Hash != hash {
		return false;
	}

	return true;
}

func validBlock(block, prevBlock *Block) bool {
	if prevBlock.Hash != block.PrevHash {
		return false;
	}

	if !block.validateHash(block.Hash) {
		return false;
	}

	if prevBlock.Position + 1 != block.Position {
		return false;
	}

	return true;
}


func writeBlock(w http.ResponseWriter, r *http.Request) {

	var checkoutItem BookCheckout;

	if err := json.NewDecoder(r.Body).Decode(&checkoutItem); err != nil {
		w.WriteHeader(http.StatusInternalServerError);
		log.Printf("could not write block: %v", err);
		w.Write([]byte("could not write block"));
		return;
	}

	log.Print(checkoutItem);
	
	BookBlockchain.AddBlock(checkoutItem);

	res, err := json.MarshalIndent(checkoutItem, "", " ");

	if err != nil {
		w.WriteHeader(http.StatusInternalServerError);
		log.Printf("could not marshal payload:  %v", err);
		w.Write([]byte("could not write block"));
		return;
	}

	w.WriteHeader(http.StatusOK);
	w.Write(res);
}

func newBook(w http.ResponseWriter, r *http.Request) {
	var book Book;

	if err := json.NewDecoder(r.Body).Decode(&book); err != nil {
		w.WriteHeader(http.StatusBadRequest)
		log.Printf("could not parse body");
		w.Write([]byte("Could not parse body"));
		return;
	}

	hash := md5.New();

	io.WriteString(hash, book.ISDN+book.PublishDate);
	book.ID = fmt.Sprintf("%x", hash.Sum(nil));
	
	res, err := json.MarshalIndent(book, "", "");

	if err != nil {
		w.WriteHeader(http.StatusInternalServerError);
		log.Printf("could not marshal payload: %v", err);
		w.Write([]byte("could not save book data"));
	}

	w.WriteHeader(http.StatusOK);
	w.Write(res);
}

func GenesisBlock() *Block {
	return CreateBlock(&Block{}, BookCheckout{IsGenesis: true}); 
}

func InitializeNewBlockChain() *Blockchain {
	return &Blockchain{[]*Block{GenesisBlock()}};

}

func getBlocks(w http.ResponseWriter, r *http.Request) {
	bytes, err := json.MarshalIndent(BookBlockchain.blocks, "", " ");

	log.Print(string(bytes));

	if err != nil {
		w.WriteHeader(http.StatusInternalServerError);
		json.NewEncoder(w).Encode(err);
		return;
	}

	io.WriteString(w, string(bytes));
}

func main() {

	BookBlockchain = InitializeNewBlockChain();

	r := mux.NewRouter();
	r.HandleFunc("/", getBlocks).Methods("GET");
	r.HandleFunc("/", writeBlock).Methods("POST");
	r.HandleFunc("/new", newBook).Methods("POST");

	go func() {
		for _, block := range BookBlockchain.blocks { 
			fmt.Printf("Prev. hash: %x\n", block.PrevHash);
			bytes, _ := json.MarshalIndent(block.Data, "", " ");
			fmt.Printf("Data: %v\n", string(bytes));
			fmt.Printf("Hash: %v\n", block.Hash);
			fmt.Println();
		}
	}()

	log.Println("Listening on PORT 3000");

	err := http.ListenAndServe(":3000", r);

	if err != nil {
		log.Fatalf("could not start server: %v", err); 
	}
} 