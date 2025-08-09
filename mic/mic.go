package mic

import (
	"fmt"
	"log"

	"github.com/linuxdeepin/go-lib/asound"
)

const (
	SND_CTL_NONBLOCK = 0x0001
	SND_CTL_ASYNC    = 0x0002
	SND_CTL_READONLY = 0x0004
	SND_CTL_BLOCK    = 0x0008

	SND_CTL_TLVT_DB_SCALE       = 0x0001 // Basic dB scale
	SND_CTL_TLVT_DB_LINEAR      = 0x0002 // Linear volume mapping
	SND_CTL_TLVT_DB_RANGE       = 0x0003 // Min/max dB range
	SND_CTL_TLVT_DB_MINMAX      = 0x0004 // dB bounds
	SND_CTL_TLVT_DB_MINMAX_MUTE = 0x0005 // dB bounds including mute
)

const (
	SND_MIXER_SCHN_UNKNOWN      = -1
	SND_MIXER_SCHN_FRONT_LEFT   = SND_MIXER_SCHN_UNKNOWN + 1
	SND_MIXER_SCHN_FRONT_RIGHT  = SND_MIXER_SCHN_FRONT_LEFT + 1
	SND_MIXER_SCHN_REAR_LEFT    = SND_MIXER_SCHN_FRONT_RIGHT + 1
	SND_MIXER_SCHN_REAR_RIGHT   = SND_MIXER_SCHN_REAR_LEFT + 1
	SND_MIXER_SCHN_FRONT_CENTER = SND_MIXER_SCHN_REAR_RIGHT + 1
	SND_MIXER_SCHN_WOOFER       = SND_MIXER_SCHN_FRONT_CENTER + 1
	SND_MIXER_SCHN_SIDE_LEFT    = SND_MIXER_SCHN_WOOFER + 1
	SND_MIXER_SCHN_SIDE_RIGHT   = SND_MIXER_SCHN_SIDE_LEFT + 1
	SND_MIXER_SCHN_REAR_CENTER  = SND_MIXER_SCHN_SIDE_RIGHT + 1
	SND_MIXER_SCHN_LAST         = 31
	SND_MIXER_SCHN_MONO         = SND_MIXER_SCHN_FRONT_LEFT
)

func EnumSoundCards() ([]*SoundCard, error) {
	soundCards := make([]*SoundCard, 0)
	var cardIdx int = -1
	for {
		err := asound.CardNext(&cardIdx)
		if err != nil {
			log.Printf("Failed to get next card. Reason: %s\n", err)
			return nil, &MicError{
				reason: err.Error(),
			}
		}

		if cardIdx == -1 {
			return soundCards, nil
		}

		device := fmt.Sprintf("hw:%d", cardIdx)
		soundCard, err := detectCard(device)
		if err != nil {
			log.Fatalf("Failed to detect card. Reason: %s\n", err)
			return nil, &MicError{
				reason: err.Error(),
			}
		}

		soundCard.cardIndex = cardIdx
		soundCards = append(soundCards, soundCard)
	}
}

// Probes the current sound card for output channels
// Returns an object that can be used to later connect back
// to the target sound card
func detectCard(device string) (*SoundCard, error) {
	ctl, err := asound.CTLOpen(device, SND_CTL_READONLY|SND_CTL_NONBLOCK)
	if err != nil {
		log.Printf("CTLOpen failed. Reason: %s\n", err)
		return nil, &MicError{
			reason: err.Error(),
		}
	}

	defer func() {
		ctl.Close()
		if err != nil {
			log.Printf("Failed to close CTL. Reason: %s\n", err)
			panic(err)
		}
	}()

	soundCard := SoundCard{
		cardId: device,
	}

	cardInfo, err := asound.NewCTLCardInfo()
	if err != nil {
		log.Printf("NewCTLCardInfo failed. Reason: %s\n", err)
		return nil, &MicError{
			reason: err.Error(),
		}
	}

	defer cardInfo.Free()

	err = ctl.CardInfo(cardInfo)
	if err != nil {
		log.Printf("CardInfo failed. Reason: %s\n", err)
		return nil, &MicError{
			reason: err.Error()}
	}

	soundCard.ShortName = cardInfo.GetName()
	soundCard.LongName = cardInfo.GetLongName()
	soundCard.MixerName = cardInfo.GetMixerName()
	soundCard.DriverName = cardInfo.GetDriver()
	log.Printf("Found sound card: %s, %s, %s\n", cardInfo.GetName(), cardInfo.GetLongName(), cardInfo.GetMixerName())

	mixer, err := asound.OpenMixer(0)
	if err != nil {
		log.Printf("OpenMixer failed. Reason: %s\n", err)
		return nil, &MicError{
			reason: err.Error(),
		}
	}

	err = mixer.Attach(device)
	if err != nil {
		log.Printf("Attach failed. Reason: %s\n", err)
		return nil, &MicError{
			reason: err.Error(),
		}
	}

	err = mixer.SelemRegister(nil, nil)
	if err != nil {
		log.Printf("SelemRegister failed. Reason: %s\n", err)
		return nil, &MicError{
			reason: err.Error(),
		}
	}

	err = mixer.Load()
	if err != nil {
		log.Printf("Load failed. Reason: %s\n", err)
		return nil, &MicError{
			reason: err.Error(),
		}
	}

	defer func() {
		err = mixer.Close()
		if err != nil {
			log.Printf("Close failed. Reason: %s\n", err)
			panic(err)
		}
	}()

	outChannels := make([]*OutputChannel, 0)
	var index uint = 0
	selem := mixer.FirstElem()
	for selem.Ptr != nil {

		// NOTE: defer should free the object as soon as possible
		func() {
			selemId, err := asound.NewMixerSelemId()
			if err != nil {
				log.Printf("NewMixerSelemId failed. Reason: %s\n", err)
				return
			}

			defer selemId.Free()

			selem.GetSelemId(selemId)

			if selem.SelemIsActive() &&
				selem.SelemHasCaptureChannel(SND_MIXER_SCHN_MONO) &&
				((!selem.SelemHasCommonVolume() && selem.SelemHasCaptureVolume()) ||
					(!selem.SelemHasCommonSwitch() && selem.SelemHasCommonVolume())) {

				selemName := selemId.GetName()

				minVolume, maxVolume := selem.SelemGetCaptureVolumeRange()

				channels := make([]int, 0)
				curVolume := make([]int, 0)
				var selemChId asound.MixerSelemChannelId = 0
				for ; selemChId != SND_MIXER_SCHN_LAST; selemChId = selemChId + 1 {
					if selem.SelemHasCaptureChannel(asound.MixerSelemChannelId(selemChId)) {
						channels = append(channels, int(selemChId))

						volume, err := selem.SelemGetCaptureVolume(selemChId)
						if err != nil {
							log.Printf("SelemGetCaptureVolume failed. Reason: %s\n", err)
							return
						}

						curVolume = append(curVolume, volume)
						log.Printf("%s has channel %d. Current volume: %d [%d:%d]",
							selemName, selemChId, volume, minVolume, maxVolume)
					}
				}

				// Only add channels we can modify
				if len(channels) != 0 && len(curVolume) != 0 {
					outChannel := OutputChannel{
						selemId:   index,
						soundCard: &soundCard,
						Name:      selemName,
						MinVolume: minVolume,
						MaxVolume: maxVolume,
						channels:  channels,
						CurVolume: curVolume[0],
					}

					outChannels = append(outChannels, &outChannel)
				} else {
					log.Printf("Output channel '%s' from card '%s' is empty. Skipping\n",
						selemName, soundCard.LongName)
				}
			}

			selem = selem.Next()
			index++
		}()
	}

	soundCard.OutputChannels = outChannels
	return &soundCard, nil
}
