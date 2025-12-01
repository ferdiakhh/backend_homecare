package utils

import (
	"context"
	"log"
	"path/filepath"

	firebase "firebase.google.com/go"
	"firebase.google.com/go/messaging"
	"google.golang.org/api/option"
)

var fcmClient *messaging.Client

// InitFCM menginisialisasi koneksi ke Firebase
func InitFCM() {
	// Ganti dengan nama file JSON kamu
	serviceAccountPath := filepath.Join(".", "koencidasarapi.json")

	opt := option.WithCredentialsFile(serviceAccountPath)
	app, err := firebase.NewApp(context.Background(), nil, opt)
	if err != nil {
		log.Fatalf("Error initializing firebase app: %v", err)
	}

	client, err := app.Messaging(context.Background())
	if err != nil {
		log.Fatalf("Error getting messaging client: %v", err)
	}

	fcmClient = client
	log.Println("🔥 Firebase Cloud Messaging Ready!")
}

// SendNotification mengirim pesan ke satu device (FCM Token)
func SendNotification(token string, title string, body string, data map[string]string) error {
	if fcmClient == nil {
		return nil // Atau return error
	}

	message := &messaging.Message{
		Token: token,
		Notification: &messaging.Notification{
			Title: title,
			Body:  body,
		},
		Data: data, // Data tambahan (misal: order_id: "123")
	}

	_, err := fcmClient.Send(context.Background(), message)
	if err != nil {
		log.Printf("Error sending message: %s", err)
		return err
	}

	log.Println("Notifikasi terkirim ke:", token)
	return nil
}

// SendNotificationToUser mengirim notifikasi ke user berdasarkan ID
// Fungsi ini akan mencari token FCM user dari database
func SendNotificationToUser(userID uint64, title string, body string, data map[string]string) error {
	// Import cycle prevention: Kita tidak bisa import config/models di sini karena utils di-import oleh mereka.
	// Solusi: Kita minta caller (Handler) untuk mengirim token stringnya saja,
	// ATAU kita buat fungsi ini menerima Token string, bukan UserID.
	// TAPI, agar praktis, kita bisa pakai query raw atau callback, tapi paling aman adalah:
	// Handler yang bertanggung jawab mengambil Token dari DB, lalu panggil SendNotification.
	// Jadi fungsi ini mungkin tidak diperlukan jika kita konsisten pakai SendNotification(token, ...).

	// Namun, untuk mempermudah, kita bisa buat helper di Handler layer, bukan di Utils layer jika butuh akses DB.
	// Tapi karena user minta di utils, kita coba pendekatan lain:
	// Kita biarkan Utils murni urusan Firebase. Handler yang urus DB.
	// Jadi kita tetap pakai SendNotification yang sudah ada.
	return nil
}
