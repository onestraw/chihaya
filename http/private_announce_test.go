// Copyright 2014 The Chihaya Authors. All rights reserved.
// Use of this source code is governed by the BSD 2-Clause license,
// which can be found in the LICENSE file.

package http

import (
	"io/ioutil"
	"net"
	"net/http"
	"net/http/httptest"
	"reflect"
	"testing"

	"github.com/chihaya/bencode"
	"github.com/chihaya/chihaya/config"
	"github.com/chihaya/chihaya/models"

	_ "github.com/chihaya/chihaya/drivers/backend/noop"
	_ "github.com/chihaya/chihaya/drivers/tracker/memory"
)

func loadTestData(tkr *Tracker) (err error) {
	conn, err := tkr.tp.Get()
	if err != nil {
		return
	}

	users := []string{
		"vvvvvvvvvvvvvvvvvvvvvvvvvvvvvvv1",
		"vvvvvvvvvvvvvvvvvvvvvvvvvvvvvvv2",
		"vvvvvvvvvvvvvvvvvvvvvvvvvvvvvvv3",
	}

	for i, passkey := range users {
		err = conn.PutUser(&models.User{
			ID:      uint64(i + 1),
			Passkey: passkey,
		})

		if err != nil {
			return
		}
	}

	err = conn.PutClient("TR2820")
	if err != nil {
		return
	}

	torrent := &models.Torrent{
		ID:       1,
		Infohash: string([]byte{0x89, 0xd4, 0xbc, 0x52, 0x11, 0x16, 0xca, 0x1d, 0x42, 0xa2, 0xf3, 0x0d, 0x1f, 0x27, 0x4d, 0x94, 0xe4, 0x68, 0x1d, 0xaf}),
		Seeders:  make(map[string]models.Peer),
		Leechers: make(map[string]models.Peer),
	}

	err = conn.PutTorrent(torrent)
	if err != nil {
		return
	}

	err = conn.PutLeecher(torrent.Infohash, &models.Peer{
		ID:        "-TR2820-vvvvvvvvvvv1",
		UserID:    1,
		TorrentID: torrent.ID,
		IP:        net.ParseIP("127.0.0.1"),
		Port:      34000,
		Left:      0,
	})
	if err != nil {
		return
	}

	err = conn.PutLeecher(torrent.Infohash, &models.Peer{
		ID:        "-TR2820-vvvvvvvvvvv3",
		UserID:    3,
		TorrentID: torrent.ID,
		IP:        net.ParseIP("::1"),
		Port:      34000,
		Left:      0,
	})
	return
}

func testRoute(cfg *config.Config, url string) ([]byte, error) {
	tkr, err := NewTracker(cfg)
	if err != nil {
		return nil, err
	}

	err = loadTestData(tkr)
	if err != nil {
		return nil, err
	}

	srv := httptest.NewServer(setupRoutes(tkr, cfg))
	defer srv.Close()

	resp, err := http.Get(srv.URL + url)
	if err != nil {
		return nil, err
	}

	body, err := ioutil.ReadAll(resp.Body)
	resp.Body.Close()
	if err != nil {
		return nil, err
	}

	return body, nil
}

func TestPrivateAnnounce(t *testing.T) {
	cfg := config.DefaultConfig
	cfg.Private = true

	url := "/users/vvvvvvvvvvvvvvvvvvvvvvvvvvvvvvv2/announce?info_hash=%89%d4%bcR%11%16%ca%1dB%a2%f3%0d%1f%27M%94%e4h%1d%af&peer_id=-TR2820-vvvvvvvvvvv2&port=51413&uploaded=0&downloaded=0&left=0&numwant=1&key=3c8e3319&compact=0"

	expected := bencode.Dict{
		"complete":     int64(1),
		"incomplete":   int64(2),
		"interval":     int64(1800),
		"min interval": int64(900),
		"peers": bencode.List{
			bencode.Dict{
				"ip":      "127.0.0.1",
				"peer id": "-TR2820-vvvvvvvvvvv1",
				"port":    int64(34000),
			},
		},
	}

	response, err := testRoute(&cfg, url)
	if err != nil {
		t.Error(err)
	}
	got, err := bencode.Unmarshal(response)

	if !reflect.DeepEqual(got, expected) {
		t.Errorf("\ngot:    %#v\nwanted: %#v", got, expected)
	}

	url = "/users/vvvvvvvvvvvvvvvvvvvvvvvvvvvvvvv2/announce?info_hash=%89%d4%bcR%11%16%ca%1dB%a2%f3%0d%1f%27M%94%e4h%1d%af&peer_id=-TR2820-vvvvvvvvvvv2&port=51413&uploaded=0&downloaded=0&left=0&numwant=2&key=3c8e3319&compact=0"

	expected = bencode.Dict{
		"complete":     int64(1),
		"incomplete":   int64(2),
		"interval":     int64(1800),
		"min interval": int64(900),
		"peers": bencode.List{
			bencode.Dict{
				"ip":      "127.0.0.1",
				"peer id": "-TR2820-vvvvvvvvvvv1",
				"port":    int64(34000),
			},
			bencode.Dict{
				"ip":      "::1",
				"peer id": "-TR2820-vvvvvvvvvvv3",
				"port":    int64(34000),
			},
		},
	}

	response, err = testRoute(&cfg, url)
	if err != nil {
		t.Error(err)
	}
	got, err = bencode.Unmarshal(response)

	if !reflect.DeepEqual(got, expected) {
		t.Errorf("\ngot:    %#v\nwanted: %#v", got, expected)
	}
}
