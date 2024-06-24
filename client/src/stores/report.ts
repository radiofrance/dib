import { writable } from 'svelte/store';
import type { Image } from '$models/dagImage.interface';

// reportDataStore list all built images.
// Read from property "window.dib_images" in "map.js" files
export const reportDataStore = writable<string[]>([]);

// imagesStore holds all built images data (build logs, tests and analysis results...)
// Read from files in "reports/{current}/data/{image-name}/*" :
// - docker.txt -> image build logs
// - goss.json  -> image tests results
// - trivy.json -> image scan results
export const imagesStore = writable<Image[]>([]);
