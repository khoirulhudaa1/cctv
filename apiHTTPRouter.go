package main

import (
	"net/http"
	"os"
	"time"

	"github.com/deepch/RTSPtoWeb/libraries"
	"github.com/gin-gonic/autotls"
	"github.com/gin-gonic/gin"
	"github.com/sirupsen/logrus"
	"golang.org/x/crypto/bcrypt"
)

// var DB *sql.DB // pastikan sudah di-setup di file init

// Message resp struct
type Message struct {
	Status  int         `json:"status"`
	Payload interface{} `json:"payload"`
}
:=
// HTTPAPIServer start http server routes
func HTTPAPIServer() {
	//Set HTTP API mode
	log.WithFields(logrus.Fields{
		"module": "http_server",
		"func":   "RTSPServer",
		"call":   "Start",
	}).Infoln("Server HTTP start")
	var public *gin.Engine
	if !Storage.ServerHTTPDebug() {
		gin.SetMode(gin.ReleaseMode)
		public = gin.New()
	} else {
		gin.SetMode(gin.DebugMode)
		public = gin.Default()
	}

	public.Use(CrossOrigin())
	//Add private login password protect methods
	privat := public.Group("/")
	if Storage.ServerHTTPLogin() != "" && Storage.ServerHTTPPassword() != "" {
		privat.Use(gin.BasicAuth(gin.Accounts{Storage.ServerHTTPLogin(): Storage.ServerHTTPPassword()}))
	}

	/*
		Static HTML Files Demo Mode
	*/

	if Storage.ServerHTTPDemo() {
		public.LoadHTMLGlob(Storage.ServerHTTPDir() + "/templates/*")

		// public.LoadHTMLGlob(Storage.ServerHTTPDir() + "/templates/*.tmpl")
		// public.StaticFS("/img", http.Dir(Storage.ServerHTTPDir()+"/templates/img"))

		// public.GET("/", HTTPAPIServerIndex)
		public.GET("/", libraries.RequireLogin(), HTTPAPIServerIndex)
		public.GET("/pages/stream/list", HTTPAPIStreamList)
		public.GET("/pages/stream/add", HTTPAPIAddStream)

		public.GET("/login", func(c *gin.Context) {
			c.HTML(http.StatusOK, "login.tmpl", nil)
		})

		public.POST("/login", func(c *gin.Context) {
			username := c.PostForm("username")
			password := c.PostForm("password")

			var hashedPassword string
			err := DB.QueryRow("SELECT password FROM users WHERE username=?", username).Scan(&hashedPassword)
			if err != nil {
				c.HTML(http.StatusUnauthorized, "login.tmpl", gin.H{"error": "Username atau password salah!"})
				return
			}

			err = bcrypt.CompareHashAndPassword([]byte(hashedPassword), []byte(password))
			if err != nil {
				c.HTML(http.StatusUnauthorized, "login.tmpl", gin.H{"error": "Username atau password salah!"})
				return
			}

			libraries.SetCookieUser(c, username)
			c.Redirect(http.StatusFound, "/")
		})

		public.GET("/users", libraries.RequireLogin(), func(c *gin.Context) {
			rows, err := DB.Query("SELECT id, username FROM users")
			if err != nil {
				c.HTML(http.StatusInternalServerError, "users.tmpl", gin.H{"error": "Gagal ambil data"})
				return
			}
			defer rows.Close()

			var users []map[string]interface{}
			for rows.Next() {
				var id int
				var username string
				rows.Scan(&id, &username)
				users = append(users, map[string]interface{}{"id": id, "username": username})
			}

			success := c.Query("success")
			errorMsg := c.Query("error")

			c.HTML(http.StatusOK, "users.tmpl", gin.H{
				"users":   users,
				"success": success,
				"error":   errorMsg,
			})
		})

		public.GET("/users/add", libraries.RequireLogin(), func(c *gin.Context) {
			c.HTML(http.StatusOK, "adduser.tmpl", nil)
		})

		public.POST("/users/add", libraries.RequireLogin(), func(c *gin.Context) {
			username := c.PostForm("username")
			password := c.PostForm("password")
			hash, _ := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
			_, err := DB.Exec("INSERT INTO users(username, password) VALUES(?, ?)", username, hash)
			if err != nil {
				c.HTML(http.StatusBadRequest, "adduser.tmpl", gin.H{"error": "Gagal tambah user, mungkin username sudah ada"})
				return
			}
			c.Redirect(http.StatusFound, "/users?success=User berhasil ditambahkan")

			// c.Redirect(http.StatusFound, "/users")
		})

		public.GET("/users/edit/:id", libraries.RequireLogin(), func(c *gin.Context) {
			id := c.Param("id")
			var username string
			err := DB.QueryRow("SELECT username FROM users WHERE id=?", id).Scan(&username)
			if err != nil {
				c.HTML(http.StatusNotFound, "edituser.tmpl", gin.H{"error": "User tidak ditemukan"})
				return
			}
			c.HTML(http.StatusOK, "edituser.tmpl", gin.H{"id": id, "username": username})
		})

		public.POST("/users/edit/:id", libraries.RequireLogin(), func(c *gin.Context) {
			id := c.Param("id")
			username := c.PostForm("username")
			password := c.PostForm("password")

			if password != "" {
				hash, _ := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
				_, err := DB.Exec("UPDATE users SET username=?, password=? WHERE id=?", username, hash, id)
				if err != nil {
					c.HTML(http.StatusBadRequest, "edituser.tmpl", gin.H{"error": "Gagal update user"})
					return
				}
			} else {
				_, err := DB.Exec("UPDATE users SET username=? WHERE id=?", username, id)
				if err != nil {
					c.HTML(http.StatusBadRequest, "edituser.tmpl", gin.H{"error": "Gagal update user"})
					return
				}
			}
			c.Redirect(http.StatusFound, "/users?success=User berhasil Di Ubah")

			// c.Redirect(http.StatusFound, "/users")
		})

		public.POST("/users/delete/:id", libraries.RequireLogin(), func(c *gin.Context) {
			id := c.Param("id")
			_, err := DB.Exec("DELETE FROM users WHERE id=?", id)
			if err != nil {
				c.HTML(http.StatusBadRequest, "users.tmpl", gin.H{"error": "Gagal hapus user"})
				return
			}
			c.Redirect(http.StatusFound, "/users?success=User berhasil Di Hapus")

			// c.Redirect(http.StatusFound, "/users")
		})

		public.GET("/logout", func(c *gin.Context) {
			libraries.ClearCookieUser(c)
			c.Redirect(http.StatusFound, "/login")
		})

		public.GET("/change-password", libraries.RequireLogin(), func(c *gin.Context) {
			c.HTML(http.StatusOK, "changepassword.tmpl", nil)
		})

		public.POST("/change-password", libraries.RequireLogin(), func(c *gin.Context) {
			username := libraries.GetCookieUser(c)
			oldPassword := c.PostForm("old_password")
			newPassword := c.PostForm("new_password")
			confirmPassword := c.PostForm("confirm_password")

			var hashedPassword string
			err := DB.QueryRow("SELECT password FROM users WHERE username=?", username).Scan(&hashedPassword)
			if err != nil {
				c.HTML(http.StatusInternalServerError, "changepassword.tmpl", gin.H{"error": "Gagal mengambil data pengguna"})
				return
			}

			// Cek password lama
			err = bcrypt.CompareHashAndPassword([]byte(hashedPassword), []byte(oldPassword))
			if err != nil {
				c.HTML(http.StatusBadRequest, "changepassword.tmpl", gin.H{"error": "Password lama salah"})
				return
			}

			// Tambahan validasi: password baru sama dengan lama
			if oldPassword == newPassword {
				c.HTML(http.StatusBadRequest, "changepassword.tmpl", gin.H{
					"error": "Password baru tidak boleh sama dengan password lama. Silakan gunakan password yang berbeda.",
				})
				return
			}

			if newPassword != confirmPassword {
				c.HTML(http.StatusBadRequest, "changepassword.tmpl", gin.H{"error": "Konfirmasi password tidak cocok"})
				return
			}

			// Hash password baru dan update
			newHashedPassword, err := bcrypt.GenerateFromPassword([]byte(newPassword), bcrypt.DefaultCost)
			if err != nil {
				c.HTML(http.StatusInternalServerError, "changepassword.tmpl", gin.H{"error": "Gagal mengenkripsi password baru"})
				return
			}

			_, err = DB.Exec("UPDATE users SET password=? WHERE username=?", newHashedPassword, username)
			if err != nil {
				c.HTML(http.StatusInternalServerError, "changepassword.tmpl", gin.H{"error": "Gagal mengupdate password"})
				return
			}

			c.HTML(http.StatusOK, "changepassword.tmpl", gin.H{"success": " Password berhasil diganti. Gunakan password baru saat login berikutnya!"})
		})

		// Ini bagian yang tadinya di luar if
		public.GET("/pages/stream/edit/:uuid", HTTPAPIEditStream)
		public.GET("/pages/player/hls/:uuid/:channel", HTTPAPIPlayHls)
		public.GET("/pages/player/mse/:uuid/:channel", HTTPAPIPlayMse)
		public.GET("/pages/player/webrtc/:uuid/:channel", HTTPAPIPlayWebrtc)
		public.GET("/pages/multiview", HTTPAPIMultiview)
		public.Any("/pages/multiview/full", HTTPAPIFullScreenMultiView)
		public.GET("/pages/documentation", HTTPAPIServerDocumentation)
		public.GET("/pages/player/all/:uuid/:channel", HTTPAPIPlayAll)
		public.StaticFS("/static", http.Dir(Storage.ServerHTTPDir()+"/static"))
		// log.Println("🟢 Server berjalan di http://localhost:8080")
		// http.ListenAndServe(":8080", public)
	}
	// <--- penutup if dipindahkan ke sini

	// Sekarang blok privat bisa dideklarasikan di luar
	privat.GET("/streams", HTTPAPIServerStreams)
	privat.POST("/stream/:uuid/add", HTTPAPIServerStreamAdd)
	privat.POST("/stream/:uuid/edit", HTTPAPIServerStreamEdit)
	privat.GET("/stream/:uuid/delete", HTTPAPIServerStreamDelete)
	privat.GET("/stream/:uuid/reload", HTTPAPIServerStreamReload)
	privat.GET("/stream/:uuid/info", HTTPAPIServerStreamInfo)

	/*
		Streams Multi Control elements
	*/

	privat.POST("/streams/multi/control/add", HTTPAPIServerStreamsMultiControlAdd)
	privat.POST("/streams/multi/control/delete", HTTPAPIServerStreamsMultiControlDelete)

	/*
		Stream Channel elements
	*/

	privat.POST("/stream/:uuid/channel/:channel/add", HTTPAPIServerStreamChannelAdd)
	privat.POST("/stream/:uuid/channel/:channel/edit", HTTPAPIServerStreamChannelEdit)
	privat.GET("/stream/:uuid/channel/:channel/delete", HTTPAPIServerStreamChannelDelete)
	privat.GET("/stream/:uuid/channel/:channel/codec", HTTPAPIServerStreamChannelCodec)
	privat.GET("/stream/:uuid/channel/:channel/reload", HTTPAPIServerStreamChannelReload)
	privat.GET("/stream/:uuid/channel/:channel/info", HTTPAPIServerStreamChannelInfo)

	/*
		Stream video elements
	*/
	//HLS
	public.GET("/stream/:uuid/channel/:channel/hls/live/index.m3u8", HTTPAPIServerStreamHLSM3U8)
	public.GET("/stream/:uuid/channel/:channel/hls/live/segment/:seq/file.ts", HTTPAPIServerStreamHLSTS)
	//HLS remote record
	//public.GET("/stream/:uuid/channel/:channel/hls/rr/:s/:e/index.m3u8", HTTPAPIServerStreamRRM3U8)
	//public.GET("/stream/:uuid/channel/:channel/hls/rr/:s/:e/:seq/file.ts", HTTPAPIServerStreamRRTS)
	//HLS LL
	public.GET("/stream/:uuid/channel/:channel/hlsll/live/index.m3u8", HTTPAPIServerStreamHLSLLM3U8)
	public.GET("/stream/:uuid/channel/:channel/hlsll/live/init.mp4", HTTPAPIServerStreamHLSLLInit)
	public.GET("/stream/:uuid/channel/:channel/hlsll/live/segment/:segment/:any", HTTPAPIServerStreamHLSLLM4Segment)
	public.GET("/stream/:uuid/channel/:channel/hlsll/live/fragment/:segment/:fragment/:any", HTTPAPIServerStreamHLSLLM4Fragment)
	//MSE
	public.GET("/stream/:uuid/channel/:channel/mse", HTTPAPIServerStreamMSE)
	public.POST("/stream/:uuid/channel/:channel/webrtc", HTTPAPIServerStreamWebRTC)
	//Save fragment to mp4
	public.GET("/stream/:uuid/channel/:channel/save/mp4/fragment/:duration", HTTPAPIServerStreamSaveToMP4)
	/*
		HTTPS Mode Cert
		# Key considerations for algorithm "RSA" ≥ 2048-bit
		openssl genrsa -out server.key 2048

		# Key considerations for algorithm "ECDSA" ≥ secp384r1
		# List ECDSA the supported curves (openssl ecparam -list_curves)
		#openssl ecparam -genkey -name secp384r1 -out server.key
		#Generation of self-signed(x509) public key (PEM-encodings .pem|.crt) based on the private (.key)

		openssl req -new -x509 -sha256 -key server.key -out server.crt -days 3650
	*/
	if Storage.ServerHTTPS() {
		if Storage.ServerHTTPSAutoTLSEnable() {
			go func() {
				err := autotls.Run(public, Storage.ServerHTTPSAutoTLSName()+Storage.ServerHTTPSPort())
				if err != nil {
					log.Println("Start HTTPS Server Error", err)
				}
			}()
		} else {
			go func() {
				err := public.RunTLS(Storage.ServerHTTPSPort(), Storage.ServerHTTPSCert(), Storage.ServerHTTPSKey())
				if err != nil {
					log.WithFields(logrus.Fields{
						"module": "http_router",
						"func":   "HTTPSAPIServer",
						"call":   "ServerHTTPSPort",
					}).Fatalln(err.Error())
					os.Exit(1)
				}
			}()
		}
	}
	err := public.Run(Storage.ServerHTTPPort())
	if err != nil {
		log.WithFields(logrus.Fields{
			"module": "http_router",
			"func":   "HTTPAPIServer",
			"call":   "ServerHTTPPort",
		}).Fatalln(err.Error())
		os.Exit(1)
	}

}

// HTTPAPIServerIndex index file
func HTTPAPIServerIndex(c *gin.Context) {
	c.HTML(http.StatusOK, "index.tmpl", gin.H{
		"port":    Storage.ServerHTTPPort(),
		"streams": Storage.Streams,
		"version": time.Now().String(),
		"page":    "index",
	})

}

func HTTPAPIServerDocumentation(c *gin.Context) {
	c.HTML(http.StatusOK, "documentation.tmpl", gin.H{
		"port":    Storage.ServerHTTPPort(),
		"streams": Storage.Streams,
		"version": time.Now().String(),
		"page":    "documentation",
	})
}

func HTTPAPIStreamList(c *gin.Context) {
	c.HTML(http.StatusOK, "stream_list.tmpl", gin.H{
		"port":    Storage.ServerHTTPPort(),
		"streams": Storage.Streams,
		"version": time.Now().String(),
		"page":    "stream_list",
	})
}

func HTTPAPIPlayHls(c *gin.Context) {
	c.HTML(http.StatusOK, "play_hls.tmpl", gin.H{
		"port":    Storage.ServerHTTPPort(),
		"streams": Storage.Streams,
		"version": time.Now().String(),
		"page":    "play_hls",
		"uuid":    c.Param("uuid"),
		"channel": c.Param("channel"),
	})
}
func HTTPAPIPlayMse(c *gin.Context) {
	c.HTML(http.StatusOK, "play_mse.tmpl", gin.H{
		"port":    Storage.ServerHTTPPort(),
		"streams": Storage.Streams,
		"version": time.Now().String(),
		"page":    "play_mse",
		"uuid":    c.Param("uuid"),
		"channel": c.Param("channel"),
	})
}
func HTTPAPIPlayWebrtc(c *gin.Context) {
	c.HTML(http.StatusOK, "play_webrtc.tmpl", gin.H{
		"port":    Storage.ServerHTTPPort(),
		"streams": Storage.Streams,
		"version": time.Now().String(),
		"page":    "play_webrtc",
		"uuid":    c.Param("uuid"),
		"channel": c.Param("channel"),
	})
}
func HTTPAPIAddStream(c *gin.Context) {
	c.HTML(http.StatusOK, "add_stream.tmpl", gin.H{
		"port":    Storage.ServerHTTPPort(),
		"streams": Storage.Streams,
		"version": time.Now().String(),
		"page":    "add_stream",
	})
}
func HTTPAPIEditStream(c *gin.Context) {
	c.HTML(http.StatusOK, "edit_stream.tmpl", gin.H{
		"port":    Storage.ServerHTTPPort(),
		"streams": Storage.Streams,
		"version": time.Now().String(),
		"page":    "edit_stream",
		"uuid":    c.Param("uuid"),
	})
}

func HTTPAPIMultiview(c *gin.Context) {
	c.HTML(http.StatusOK, "multiview.tmpl", gin.H{
		"port":    Storage.ServerHTTPPort(),
		"streams": Storage.Streams,
		"version": time.Now().String(),
		"page":    "multiview",
	})
}

func HTTPAPIPlayAll(c *gin.Context) {
	c.HTML(http.StatusOK, "play_all.tmpl", gin.H{
		"port":    Storage.ServerHTTPPort(),
		"streams": Storage.Streams,
		"version": time.Now().String(),
		"page":    "play_all",
		"uuid":    c.Param("uuid"),
		"channel": c.Param("channel"),
	})
}

type MultiViewOptions struct {
	Grid   int                             `json:"grid"`
	Player map[string]MultiViewOptionsGrid `json:"player"`
}
type MultiViewOptionsGrid struct {
	UUID       string `json:"uuid"`
	Channel    int    `json:"channel"`
	PlayerType string `json:"playerType"`
}

func HTTPAPIFullScreenMultiView(c *gin.Context) {
	var createParams MultiViewOptions
	err := c.ShouldBindJSON(&createParams)
	if err != nil {
		log.WithFields(logrus.Fields{
			"module": "http_page",
			"func":   "HTTPAPIFullScreenMultiView",
			"call":   "BindJSON",
		}).Errorln(err.Error())
	}
	log.WithFields(logrus.Fields{
		"module": "http_page",
		"func":   "HTTPAPIFullScreenMultiView",
		"call":   "Options",
	}).Debugln(createParams)
	c.HTML(http.StatusOK, "fullscreenmulti.tmpl", gin.H{
		"port":    Storage.ServerHTTPPort(),
		"streams": Storage.Streams,
		"version": time.Now().String(),
		"options": createParams,
		"page":    "fullscreenmulti",
		"query":   c.Request.URL.Query(),
	})
}

// CrossOrigin Access-Control-Allow-Origin any methods
func CrossOrigin() gin.HandlerFunc {
	return func(c *gin.Context) {
		c.Writer.Header().Set("Access-Control-Allow-Origin", "*")
		c.Writer.Header().Set("Access-Control-Allow-Credentials", "true")
		c.Writer.Header().Set("Access-Control-Allow-Headers", "Content-Type, Content-Length, Accept-Encoding, X-CSRF-Token, Authorization, accept, origin, Cache-Control, X-Requested-With")
		c.Writer.Header().Set("Access-Control-Allow-Methods", "POST, OPTIONS, GET, PUT, DELETE")
		if c.Request.Method == "OPTIONS" {
			c.AbortWithStatus(204)
			return
		}
		c.Next()
	}
}
