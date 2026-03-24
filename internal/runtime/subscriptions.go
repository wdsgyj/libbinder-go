package runtime

// SubscriptionSet tracks runtime subscriptions such as death notifications.
type SubscriptionSet struct{}

func NewSubscriptionSet() *SubscriptionSet {
	return &SubscriptionSet{}
}
