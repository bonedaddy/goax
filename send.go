package main

import (
	"bytes"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"io/ioutil"
	"log"
	"os"
	"path"

	"github.com/rakoo/goax/pkg/ratchet"

	"golang.org/x/crypto/openpgp/armor"
)

func send(peer string) {
	r, err := openRatchet(peer)
	if err != nil {
		if err == errNoRatchet {
			fmt.Fprintf(os.Stderr, "No ratchet for %s, please send this to the peer and \"receive\" what they send you back", peer)
			fmt.Println("\n")
			r, err := createRatchet(peer)
			if err != nil {
				log.Fatalf("Couldn't create ratchet for %s: %s", peer, err)
			}
			err = saveRatchet(r, peer)
			if err != nil {
				log.Fatal("Couldn't save ratchet, will have to try another time", err)
			}
			sendRatchet(r)
			fmt.Println("")
			return
		} else {
			log.Fatal(err)
		}
	}

	fmt.Println("")
	msg, err := ioutil.ReadAll(os.Stdin)
	if err != nil {
		log.Fatal("Couldn't read all stdin")
	}
	fmt.Println("")
	cipherText, err := r.Encrypt(msg)
	if err != nil {
		if err == ratchet.ErrHandshakeNotComplete {
			fmt.Fprintf(os.Stderr, "\nSorry, the handshake is not complete yet; you can't send any messages. Please ask %s for their key exchange material and use \"goax receive %s\" to finish handshake.\n", peer, peer)
			fmt.Fprintf(os.Stderr, "Here's your own key exchange material, in case you want to send it again to them:\n\n")
			sendRatchet(r)
			os.Exit(1)
		} else {
			log.Fatal(err)
		}
	}

	if err := saveRatchet(r, peer); err != nil {
		log.Println("Couldn't save ratchet:", err)
		os.Remove(path.Join("ratchets", hex.EncodeToString([]byte(peer))))
		os.Exit(1)
	}

	if isNew(peer) {
		sendRatchet(r)
	}

	encoder, err := armor.Encode(os.Stdout, ENCRYPTED_MESSAGE_TYPE, nil)
	if err != nil {
		log.Fatal("Couldn't create armor encoder: ", err)
	}

	io.Copy(encoder, bytes.NewReader(cipherText))
	encoder.Close()
	fmt.Println("")
}

func sendRatchet(r *ratchet.Ratchet) {
	kx, err := r.GetKeyExchangeMaterial()
	if err != nil {
		log.Fatal("Couldn't get key exchange material ", err)
	}
	encoder, err := armor.Encode(os.Stdout, KEY_EXCHANGE_TYPE, nil)
	if err != nil {
		log.Fatal("Couldn't get armor encoder")
	}

	json.NewEncoder(encoder).Encode(kx)
	encoder.Close()
	fmt.Println("")
}
