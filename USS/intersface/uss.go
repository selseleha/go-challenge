package intersface

type UserSegmentationService interface {
	AddSegment(userId int64, tag string) error
}

type ImplAddSegment struct{}

func (ImplAddSegment) AddSegment(userId int64, tag string) error {
	//TODO implement me
	panic("implement me")
}
