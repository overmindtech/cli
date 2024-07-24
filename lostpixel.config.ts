import { CustomProjectConfig } from 'lost-pixel';

export const config: CustomProjectConfig = {
    customShots: {
        currentShotsPath: "./e2e",
    },

    // Lost Pixel Platform (to use in Platform mode, comment out the OSS mode and uncomment this part )
    lostPixelProjectId: 'cljcuqix95y2ui10ekiwatpau',
    apiKey: process.env.LOST_PIXEL_API_KEY,
};
