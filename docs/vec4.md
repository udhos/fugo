
The 4th component is automatically expanded to 1.0 when it is absent.

That is to say, if you pass a 3-dimensional vertex attribute pointer to a 4-dimensional vector, GL will fill-in W with 1.0 for you. I always go with this route, it avoids having to explicitly write vec4 (...) when doing matrix multiplication on the position and it also avoids wasting memory storing the 4th component.

This works for 2D coordinates too, by the way. A 2D coordinate passed to a vec4 attribute becomes vec4 (x, y, 0.0, 1.0). The general rule is this: all missing components are replaced with 0.0 except for W, which is replaced with 1.0.

https://stackoverflow.com/questions/18935203/shader-position-vec4-or-vec3
