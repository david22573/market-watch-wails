import { mount } from "svelte";
import "./style.css";
import App from "./App.svelte";

// Svelte 5 way of starting the app
const app = mount(App, {
    target: document.getElementById("app"),
});

export default app;
