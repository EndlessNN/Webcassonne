import type { Rotation } from '../api/types'

// US-10: поворот на 90° за годинниковою стрілкою; після 270 — знову 0.
export function nextRotation(r: Rotation): Rotation {
  switch (r) {
    case 0:
      return 90
    case 90:
      return 180
    case 180:
      return 270
    case 270:
      return 0
  }
}
