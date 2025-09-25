/* SPDX-License-Identifier: BSD-2-Clause */
/*
 * Copyright (C) 2020, Raspberry Pi (Trading) Ltd.
 *
 * rpicam_vid.cpp - libcamera video record app.
 */

#include "rpicam_api.h"
#include "core/rpicam_encoder.hpp"

using namespace std::placeholders;

static CameraOutputReadyCallback g_yuv420_cb = NULL;

static void outputReady(void *mem, size_t size, int64_t timestamp_us, bool keyframe)
{
		if (NULL != g_yuv420_cb)
			g_yuv420_cb((unsigned char*)mem, size);
};

int startCamera(struct CameraParams *params)
{
	RPiCamEncoder app;
	VideoOptions *options = app.GetOptions();
	
	const char *argv[] = {};
	if (options->Parse(0, (char**)argv))
	{
		g_yuv420_cb = params->cb_yuv420;

		options->Set().codec = "yuv420";
		options->Set().nopreview = true;
		options->Set().width = params->width;
		options->Set().height = params->height;
		options->Set().framerate = params->framerate;

		if (params->loglevel >= 2)
			options->Get().Print();

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
				return 0;
			else if (msg.type != RPiCamEncoder::MsgType::RequestComplete)
				throw std::runtime_error("unrecognised message!");

			CompletedRequestPtr &completed_request = std::get<CompletedRequestPtr>(msg.payload);
			app.EncodeBuffer(completed_request, app.VideoStream());
		}
	}

	return 0;
};
