import { writable } from 'svelte/store';

// reportDataStore list all built images.
// Read from property "window.dib_images" in "map.js" files
export const reportDataStore = writable([]);

// imagesStore holds all built images data (build logs, tests and analysis results...)
// Read from files in "reports/{current}/data/{image-name}/*" :
// - docker.txt -> image build logs
// - goss.json  -> image tests results
// - trivy.json -> image scan results
export const imagesStore = writable([]);
