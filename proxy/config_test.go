package proxy_test

import (
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	. "github.com/thegreatape/betamax/proxy"
	"io/ioutil"
	"net/http"

	"os"
	"path"
)

var _ = Describe("Config", func() {
	cassetteDir := path.Join(os.TempDir(), "cassettes")

	readCassette := func(name string) (data string, err error) {
		cassette, err := ioutil.ReadFile(path.Join(cassetteDir, name+".json"))
		return string(cassette), err
	}

	BeforeEach(func() {
		os.RemoveAll(cassetteDir)
	})

	It("stores bodies of requests and responses with text content types as plain text", func() {
		request := RecordedRequest{
			Header: http.Header{"Content-Type": []string{"text/plain"}},
			Body:   []byte("hello!"),
		}
		response := RecordedResponse{
			Header: http.Header{"Content-Type": []string{"text/plain"}},
			Body:   []byte("goodbye!"),
		}
		episode := Episode{Request: request, Response: response}

		config := Config{
			Cassette:    "test",
			CassetteDir: cassetteDir,
			Episodes:    []Episode{episode},
		}
		config.Save()

		cassetteJSON, err := readCassette("test")
		Expect(err).To(BeNil())
		Expect(cassetteJSON).ToNot(BeNil())
		Expect(cassetteJSON).To(MatchRegexp(`"Body": "hello!"`))
		Expect(cassetteJSON).To(MatchRegexp(`"Body": "goodbye!"`))
	})

	It("stores bodies of requests and responses with non-text content types as base64 encoded strings", func() {
		request := RecordedRequest{
			Header: http.Header{"Content-Type": []string{"image/jpg"}},
			Body:   []byte("hello!"),
		}
		response := RecordedResponse{
			Header: http.Header{"Content-Type": []string{"image/jpg"}},
			Body:   []byte("goodbye!"),
		}
		episode := Episode{Request: request, Response: response}

		config := Config{
			Cassette:    "test",
			CassetteDir: cassetteDir,
			Episodes:    []Episode{episode},
		}
		config.Save()

		cassetteJSON, err := readCassette("test")
		Expect(err).To(BeNil())
		Expect(cassetteJSON).ToNot(BeNil())
		Expect(cassetteJSON).To(MatchRegexp(`"Body": "aGVsbG8h"`))
		Expect(cassetteJSON).To(MatchRegexp(`"Body": "Z29vZGJ5ZSE="`))
	})

})
