package mic

import (
	"log"

	"github.com/linuxdeepin/go-lib/asound"
)

type MicError struct {
	reason string
}

func (e *MicError) Error() string {
	return e.reason
}

type SoundCard struct {
	ShortName      string // Card short Name
	LongName       string // Card full name
	MixerName      string // A more descriptive card name
	DriverName     string // Driver name
	OutputChannels []OutputChannel
	cardIndex      int    // Card index when enumerated
	cardId         string // Used for connecting to asound API
}

type OutputChannel struct {
	selemId              uint // SelemId
	Name                 string
	MinVolume, MaxVolume int
	CurVolume            []int
	soundCard            *SoundCard
	channels             []int
}

func (ch *OutputChannel) SetVolume(volume int) (bool, error) {
	changeNeeded := false
	for _, v := range ch.CurVolume {
		if v != volume {
			changeNeeded = true
			break
		}
	}

	if !changeNeeded {
		return false, nil
	}

	ctl, err := asound.CTLOpen(ch.soundCard.cardId, SND_CTL_READONLY|SND_CTL_NONBLOCK)
	if err != nil {
		return false, err
	}

	defer func() {
		ctl.Close()
		if err != nil {
			panic(err)
		}
	}()

	mixer, err := asound.OpenMixer(0)
	if err != nil {
		return false, err
	}

	err = mixer.Attach(ch.soundCard.cardId)
	if err != nil {
		return false, err
	}

	err = mixer.SelemRegister(nil, nil)
	if err != nil {
		return false, err
	}

	err = mixer.Load()
	if err != nil {
		return false, err
	}

	defer func() {
		err = mixer.Close()
		if err != nil {
			panic(err)
		}
	}()

	selem := mixer.FirstElem()
	for selem.Ptr != nil {

		selemId, err := asound.NewMixerSelemId()
		if err != nil {
			return false, err
		}

		defer selemId.Free()

		selem.GetSelemId(selemId)

		if selemId.GetIndex() == ch.selemId {
			wasSet := false
			for _, elemCh := range ch.channels {
				elemChId := asound.MixerSelemChannelId(elemCh)
				curVolume, err := selem.SelemGetCaptureVolume(elemChId)
				if err != nil {
					return false, err
				}

				// If we are out of sync with the volume due to OS change,
				// we wont show error, just dont do anything
				if curVolume != volume {
					err = selem.SelemSetCaptureVolume(elemChId, volume)
					if err != nil {
						return false, err
					}
				}

				wasSet = true
			}

			return wasSet, nil
		}
	}

	return false, nil
}
