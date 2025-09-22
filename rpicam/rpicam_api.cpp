/* SPDX-License-Identifier: BSD-2-Clause */
/*
 * Copyright (C) 2020, Raspberry Pi (Trading) Ltd.
 *
 * rpicam_vid.cpp - libcamera video record app.
 */

#include "rpicam_api.h"
#include "rpicam-apps/core/rpicam_encoder.hpp"

using namespace std::placeholders;

static CameraOutputReadyCallback g_cb_info = NULL;

static void outputReady(void *mem, size_t size, int64_t timestamp_us, bool keyframe)
{
		if (NULL != g_cb_info)
			g_cb_info((char*)mem, size);
};

// The main even loop for the application.
static void event_loop(RPiCamEncoder &app)
{
	app.SetEncodeOutputReadyCallback(outputReady);

	app.OpenCamera();
	app.ConfigureVideo(RPiCamEncoder::FLAG_VIDEO_JPEG_COLOURSPACE);
	app.StartEncoder();
	app.StartCamera();

	while(true)
	{
		RPiCamEncoder::Msg msg = app.Wait();
		if (msg.type == RPiCamApp::MsgType::Timeout)
		{
			LOG_ERROR("ERROR: Device timeout detected, attempting a restart!!!");
			app.StopCamera();
			app.StartCamera();
			continue;
		}

		if (msg.type == RPiCamEncoder::MsgType::Quit)
			return;
		else if (msg.type != RPiCamEncoder::MsgType::RequestComplete)
			throw std::runtime_error("unrecognised message!");

		CompletedRequestPtr &completed_request = std::get<CompletedRequestPtr>(msg.payload);
		app.EncodeBuffer(completed_request, app.VideoStream());
	}
}

int startCamera(CameraOutputReadyCallback cb_info)
{
	g_cb_info = cb_info;

	RPiCamEncoder app;
	VideoOptions *options = app.GetOptions();
	
	const char *argv[] = {
		"--width", "1280",
		"--height", "720",
		"--framerate", "30",
		"--codec", "yuv420",
		"--circular"
	};

	if (options->Parse(sizeof(argv), (char**)argv))
	{
		// This is a forever loop
		event_loop(app);
	}

	// For now, we get here if we cannot parse options
	return -1;
};
