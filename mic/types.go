package mic

import (
	"github.com/linuxdeepin/go-lib/asound"
)

type MicError struct {
	reason string
}

func (e *MicError) Error() string {
	return e.reason
}

type SoundCard struct {
	ShortName      string          `json:"shortName"` // Card short Name
	LongName       string          `json:"longName"`  // Card full name
	MixerName      string          `json:"mixerName"` // A more descriptive card name
	DriverName     string          `json:"-"`         // Driver name
	OutputChannels []OutputChannel `json:"outChannels"`
	cardIndex      int             `json:"-"` // Card index when enumerated
	cardId         string          `json:"-"` // Used for connecting to asound API
}

type OutputChannel struct {
	selemId   uint       `json:"-"` // SelemId
	Name      string     `json:"name"`
	MinVolume int        `json:"minVolume"`
	MaxVolume int        `json:"maxVolume"`
	CurVolume int        `json:"curVolume"`
	soundCard *SoundCard `json:"-"`
	channels  []int      `json:"-"`
}

func (ch *OutputChannel) SetVolume(volume int) (bool, error) {
	if ch.CurVolume == volume {
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
