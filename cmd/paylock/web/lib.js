import { h, render } from 'https://esm.sh/preact@10.25.4';
import { useState, useEffect, useRef, useCallback } from 'https://esm.sh/preact@10.25.4/hooks';
import { signal, computed, effect, batch } from 'https://esm.sh/@preact/signals@1.3.1?deps=preact@10.25.4';
import htm from 'https://esm.sh/htm@3.1.1';

const html = htm.bind(h);

export { h, render, html, useState, useEffect, useRef, useCallback, signal, computed, effect, batch };
