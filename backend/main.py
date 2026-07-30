"""KPI-дашборд отдела полиции — FastAPI backend."""

from fastapi import FastAPI
from fastapi.middleware.cors import CORSMiddleware

from routers.auth import router as auth_router
from routers.catalog import router as catalog_router
from routers.dashboard import router as dashboard_router
from routers.entries import router as entries_router
from routers.messenger import router as messenger_router
from routers.notifications import router as notifications_router
from routers.targets import router as targets_router
from seed import seed

app = FastAPI(title="KPI отдела полиции", version="1.0")

app.add_middleware(
    CORSMiddleware,
    allow_origins=["*"],
    allow_credentials=True,
    allow_methods=["*"],
    allow_headers=["*"],
)

app.include_router(auth_router)
app.include_router(catalog_router)
app.include_router(targets_router)
app.include_router(entries_router)
app.include_router(dashboard_router)
app.include_router(notifications_router)
app.include_router(messenger_router)


@app.on_event("startup")
def on_startup():
    seed()
