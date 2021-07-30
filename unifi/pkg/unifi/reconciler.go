package unifi

import (
	"context"
	"time"

	"github.com/go-logr/logr"
	"github.com/paultyng/go-unifi/unifi"
	unifiv1 "github.com/thebsdbox/kubernetes-controllers/unifi/api/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

// Reconcile will handle the "polling" of unifi objects
func Reconcile(log logr.Logger, uClient *unifi.Client, kClient client.Client) error {

	for {
		var users unifiv1.UserList

		err := kClient.List(context.TODO(), &users, &client.ListOptions{})
		if err != nil {
			log.Error(err, "")
		}
		unifiUsers, err := uClient.ListUser(context.TODO(), "default")
		if err != nil {
			log.Error(err, "")
		}
		for x := range unifiUsers {
			userIP, err := uClient.GetUserByMAC(context.TODO(), "default", unifiUsers[x].MAC)
			if err != nil {
				log.Error(err, "")
			} else {
				// Make sure that the Address exists other wise we can't create the Kubernetes Object
				if userIP.IP != "" {
					newUser := unifiv1.User{
						ObjectMeta: v1.ObjectMeta{
							Name:      userIP.IP,
							Namespace: "default",
						},
						Spec: unifiv1.UserSpec{
							MAC:      unifiUsers[x].MAC,
							IP:       userIP.IP,
							Hostname: unifiUsers[x].Hostname,
							Name:     unifiUsers[x].Name,
							LastSeen: time.Unix(int64(unifiUsers[x].LastSeen), 0).String(), // TODO - this seems wrong
						},
					}

					err = kClient.Create(context.TODO(), &newUser, &client.CreateOptions{})
					if err != nil {
						if !errors.IsAlreadyExists(err) {
							log.Error(err, "")
						}
					}
				}
			}
		}
		// sleep before reconciling
		time.Sleep(time.Second * 5)
	}
	//return nil
}
